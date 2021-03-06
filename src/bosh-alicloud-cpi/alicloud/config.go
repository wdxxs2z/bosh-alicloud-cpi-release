/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package alicloud

import (
	"bosh-alicloud-cpi/registry"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type InnerType string

const (
	DefaultClassOSSInnerEndpoint = "oss-cn-hangzhou-internal"
	DefaultVpcOSSInnerEndpoint   = "oss-cn-hangzhou-internal"
	OSSSuffix                    = "oss-"

	InnerVpc       = InnerType("VPC")
	InnerClassic   = InnerType("CLASSIC")
	PingMethod     = "tcp"
	TimeoutSeconds = 5
)

type CloudConfigJson struct {
	Root CloudConfig `json:"cloud"`
}

type CloudConfig struct {
	Plugin     string `json:"plugin"`
	Properties Config `json:"properties"`
}

type Config struct {
	OpenApi  OpenApi        `json:"alicloud"`
	Registry RegistryConfig `json:"registry"`
	Agent    AgentConfig    `json:"agent"`
}

const (
	UseForceStop = true

	WaitTimeout  = time.Duration(180) * time.Second
	WaitInterval = time.Duration(5) * time.Second

	DefaultEipWaitSeconds = 120
	DefaultSlbWeight      = 100
)

type OpenApi struct {
	Region           string `json:"region"`
	AvailabilityZone string `json:"availability_zone"`
	AccessEndpoint   string `json:"access_endpoint"`
	AccessKeyId      string `json:"access_key_id"`
	AccessKeySecret  string `json:"access_key_secret"`
	SecurityToken    string `json:"security_token"`
	Encrypted        bool   `json:"encrypted"`
}

type RegistryConfig struct {
	User     string      `json:"user"`
	Password string      `json:"password"`
	Protocol string      `json:"protocol"`
	Host     string      `json:"host"`
	Port     json.Number `json:"port"`
}

type AgentConfig struct {
	Ntp       []string        `json:"ntp"`
	Mbus      string          `json:"mbus"`
	Blobstore BlobstoreConfig `json:"blobstore"`
}

type BlobstoreConfig struct {
	Provider string                 `json:"provider"`
	Options  map[string]interface{} `json:"options"`
}

func (c Config) Validate() error {
	if c.OpenApi.GetRegion("") == "" {
		return fmt.Errorf("region can't be empty")
	}

	_, err := c.Registry.Port.Int64()
	if err != nil {
		return fmt.Errorf("bad registry.port %s", c.Registry.Port.String())
	}

	//TODO: validate more
	return nil
}

func (a OpenApi) GetRegion(region string) string {
	if region != "" {
		return region
	}
	return a.Region
}

func (a OpenApi) GetAvailabilityZone() string {
	return a.AvailabilityZone
}

func (a RegistryConfig) IsEmpty() bool {
	if a.Host == "" {
		return true
	} else {
		return false
	}
}

func NewConfigFromFile(configFile string, fs boshsys.FileSystem) (Config, error) {
	var config Config

	if configFile == "" {
		return config, bosherr.Errorf("Must provide a config file")
	}

	bytes, err := fs.ReadFile(configFile)
	if err != nil {
		return config, bosherr.WrapErrorf(err, "Reading config file '%s'", configFile)
	}

	return NewConfigFromBytes(bytes)
}

func NewConfigFromBytes(bytes []byte) (Config, error) {
	var ccs CloudConfigJson
	var config Config

	err := json.Unmarshal(bytes, &ccs)
	if err != nil {
		return config, bosherr.WrapError(err, "unmarshal config json failed")
	}

	config = ccs.Root.Properties

	err = config.Validate()
	if err != nil {
		return config, bosherr.WrapError(err, "validate config failed")
	}

	return config, nil
}

func (a RegistryConfig) ToInstanceUserData() string {
	endpoint := a.GetEndpoint()
	json := fmt.Sprintf(`{"registry":{"endpoint":"%s"}}`, endpoint)
	return json
}

func (a RegistryConfig) GetEndpoint() string {
	port, _ := a.Port.Int64()
	return fmt.Sprintf("%s://%s:%s@%s:%d", a.Protocol, a.User, a.Password, a.Host, port)
}

func (a BlobstoreConfig) AsRegistrySettings() registry.BlobstoreSettings {
	return registry.BlobstoreSettings{
		Provider: a.Provider,
		Options:  a.Options,
	}
}

func (c Config) NewEcsClient(region string) (*ecs.Client, error) {
	// Obsoleted
	client, err := ecs.NewClientWithOptions(c.OpenApi.GetRegion(region), getSdkConfig(), c.getAuthCredential(true))
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Initiating ECS Client in '%s' got an error.", c.OpenApi.GetRegion(region))
	}
	return client, nil
}

func (c Config) NewSlbClient(region string) (*slb.Client, error) {
	// Obsoleted
	client, err := slb.NewClientWithOptions(c.OpenApi.GetRegion(region), getSdkConfig(), c.getAuthCredential(true))
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Initiating SLB Client in '%s' got an error.", c.OpenApi.GetRegion(region))
	}
	return client, nil
}

func (c Config) GetRegistryClient(logger boshlog.Logger) registry.Client {
	if !c.Registry.IsEmpty() {
		return c.GetHttpRegistryClient(logger)
	} else {
		return NewRegistryManager(c, logger)
	}
}

func (c Config) NewOssClient(region string, inner bool) *oss.Client {
	ossClient, _ := oss.New(c.GetAvailableOSSEndPoint(region, inner), c.OpenApi.AccessKeyId, c.OpenApi.AccessKeySecret)
	return ossClient
}

func (c Config) GetAvailableOSSEndPoint(region string, inner bool) string {
	return "https://" + c.GetOSSEndPoint(region, inner) + ".aliyuncs.com"
}

func (c Config) GetOSSEndPoint(region string, inner bool) string {
	timeOut := time.Duration(TimeoutSeconds) * time.Second
	ep := GetOSSEndPoint(string(c.OpenApi.GetRegion(region)), "")
	if !inner {
		return ep
	}

	ep = GetOSSEndPoint("", InnerVpc)
	if _, err := net.DialTimeout(PingMethod, ep, timeOut); err != nil {
		fmt.Printf("Ping oss inner vpc endpoint %s ok", ep)
		return ep
	}

	ep = GetOSSEndPoint("", InnerClassic)
	if _, err := net.DialTimeout(PingMethod, ep, timeOut); err != nil {
		fmt.Printf("Ping oss inner ecs endpoint %s ok", ep)
		return ep
	}

	ep = GetOSSEndPoint(string(c.OpenApi.GetRegion(region)), "")
	return ep
}

// types allows ["VPC", "CLASSIC"], then return inner endpoint
// otherwise return endpoint by region
func GetOSSEndPoint(region string, types InnerType) string {
	if types == InnerVpc {
		return DefaultVpcOSSInnerEndpoint
	}

	if types == InnerClassic {
		return DefaultClassOSSInnerEndpoint
	}

	if strings.HasPrefix(region, OSSSuffix) {
		return region
	}
	return OSSSuffix + region
}

func (c Config) GetHttpRegistryClient(logger boshlog.Logger) registry.Client {
	r := c.Registry

	port, _ := r.Port.Int64()
	clientOptions := registry.ClientOptions{
		Protocol: r.Protocol,
		Host:     r.Host,
		Port:     int(port),
		Username: r.User,
		Password: r.Password,
	}

	client := registry.NewHTTPClient(clientOptions, logger)
	return client
}

func (c Config) getAuthCredential(stsSupported bool) auth.Credential {
	if stsSupported {
		return credentials.NewStsTokenCredential(c.OpenApi.AccessKeyId, c.OpenApi.AccessKeySecret, c.OpenApi.SecurityToken)
	}

	return credentials.NewAccessKeyCredential(c.OpenApi.AccessKeyId, c.OpenApi.AccessKeySecret)
}

func (c Config) GetInstanceRegion(instanceId string) (region string, err error) {
	client, err := c.NewEcsClient("")
	if err != nil {
		return
	}

	args := ecs.CreateDescribeInstanceAttributeRequest()
	args.InstanceId = instanceId

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		inst, err := client.DescribeInstanceAttribute(args)
		if err != nil {
			return bosherr.WrapErrorf(err, "Describe Instance %s Attribute in '%s' got an error.", instanceId, c.OpenApi.GetRegion(region))
		}
		if inst != nil {
			region = inst.RegionId
		}
		return nil
	})
	return
}

func (c Config) GetCrossRegions() (regions []string, err error) {
	regionMap := make(map[string]string)
	regionstr := os.Getenv("CROSS_REGIONS")
	if len(strings.TrimSpace(regionstr)) > 0 {
		for _, r := range strings.Split(strings.TrimSpace(regionstr), ",") {
			r = strings.TrimSpace(r)
			if r == c.OpenApi.GetRegion("") {
				continue
			}
			if _, ok := regionMap[r]; ok {
				continue
			}
			regions = append(regions, r)
			regionMap[r] = r
		}
	}

	client, err := c.NewEcsClient("")
	if err != nil {
		return
	}

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		resp, err := client.DescribeRegions(ecs.CreateDescribeRegionsRequest())
		if err != nil {
			return bosherr.WrapErrorf(err, "Describe Regions got an error.")
		}
		if resp != nil && len(resp.Regions.Region) > 0 {
			for _, r := range resp.Regions.Region {
				if r.RegionId == c.OpenApi.GetRegion("") {
					continue
				}
				if strings.HasPrefix(r.RegionId, "cn-") {
					if _, ok := regionMap[r.RegionId]; ok {
						continue
					}
					regions = append(regions, r.RegionId)
					regionMap[r.RegionId] = r.RegionId
				}
			}
		}
		return nil
	})
	return
}
