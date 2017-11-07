# Install cf with `cf-deployment`

## Prepare you Alibaba Cloud Environment

- Select a region get `region`
- Create a vpc
- Select a zone get `zone`
- Create a vswitch in your `zone` and get `vswtich_id`
- Create security group get `security_group_id`
- Create user access key, get `access_key_id/access_key_secret`
- Create a jumpbox vm

## Install Bosh

```
$ git clone https://github.com/aliyun/bosh-deployment.git
$ cd bosh-deployment
$ git checkout alicloud
```

use this command, modify the parameters

- vswitch_id
- security_group_id
- access_key_id
- access_key_secret
- region
- zone


export your BOSH_DIRECTOR_IP

```
export BOSH_DIRECTOR_IP=...
```

```
bosh create-env bosh-deployment/bosh.yml --state=state.json \
 --vars-store=creds.yml \
 -o bosh-deployment/alicloud/cpi.yml \
 -o bosh-deployment/jumpbox-user.yml \
 -o bosh-deployment/misc/powerdns.yml \
 -v dns_recursor_ip=8.8.8.8 \
 -v director_name=my-bosh \
 -v internal_cidr=192.168.0.0/24 \
 -v internal_gw=192.168.0.1 \
 -v internal_ip=$BOSH_DIRECTOR_IP \
 -v vswitch_id=... \
 -v security_group_id=... \
 -v access_key_id=... \
 -v access_key_secret=... \
 -v region=... \
 -v zone=...
```

## Login to Bosh


```
bosh int ./creds.yml --path /director_ssl/ca > ca-cert
bosh alias-env my-bosh -e $BOSH_DIRECTOR_IP --ca-cert ca-cert
export BOSH_ENVIRONMENT=$BOSH_DIRECTOR_IP
export BOSH_CLIENT=admin
export BOSH_CLIENT_SECRET=`bosh int ./creds.yml --path /admin_password`
export BOSH_CA_CERT=`bosh int ./creds.yml --path /director_ssl/ca`
bosh -e my-bosh login
```

You can use jumpbox.key to ssh into bosh-director

```
bosh int creds.yml --path /jumpbox_ssh/private_key > jumpbox.key
chmod 600 jumpbox.key
ssh jumpbox@${BOSH_DIRECTOR_IP} -i jumpbox.key
```

## Prepare Cloud Foundry Environment

- Select 3 availability zones
- Create vswitch in each zone get `vswitch_id`, `zone_id`, `internal_cidr`, `internal_gateway`
- Config VPC SNAT with each vswitch to enable vm internet access
- Create a Http SLB and get `http_slb_id`
- Create a domain name wild bind to slb ip. Example: config *.hello-cf.cc to 47.47.47.47
    - You can use 47.47.47.47.xip.io instead custom DNS, but it's not very stable.
- create a Tcp slb get `tcp_slb_id` [optional]


Base your previous settings, modify `bosh-deployment/alicloud/cloud-config.yml` in `bosh-deployment/alicloud/cloud-config.yml`, and update-cloud-config

```
bosh -e my-bosh update-cloud-config bosh-deployment/alicloud/cloud-config.yml
```

## Install Cloud Foundry

Get `cf-deployment`

```
$ git clone https://github.com/cloudfoundry/cf-deployment.yml
```

Upload stemcell, you can download stemcell from [here](http://bosh-alicloud.oss-cn-hangzhou.aliyuncs.com/light-bosh-stemcell-1009-alicloud-kvm-ubuntu-trusty-go_agent.tgz)

```
bosh -e my-bosh upload-stemcell light-bosh-stemcell-1009-alicloud-kvm-ubuntu-trusty-go_agent.tgz
```

modify `stemcells` section in `cf-deployment.yml`

```yaml
...
stemcells:
- alias: default
  name: bosh-alicloud-kvm-ubuntu-trusty-go_agent
  version: 1009
```

Upload cf releases (important in China regions, skip it in oversea region)
Install maybe very slow in China regions, so you can upload your releases first,
Get releases from [cf-release-277](http://bosh-alicloud.oss-cn-hangzhou.aliyuncs.com/cf-release-278.zip) and unzip it.


```
bosh upload-release binary-buildpack-release-1.0.14.tgz
bosh upload-release capi-release-1.44.0.tgz
bosh upload-release cf-mysql-release-36.7.0.tgz
bosh upload-release cf-networking-release-1.8.1.tgz
bosh upload-release cf-routing-release-0.166.0.tgz
bosh upload-release cf-smoke-tests-release-39.tgz
bosh upload-release cf-syslog-drain-release-3.tgz
bosh upload-release cflinuxfs2-release-1.165.0.tgz
bosh upload-release consul-release-190.tgz
bosh upload-release diego-release-1.29.1.tgz
bosh upload-release dotnet-core-buildpack-release-1.0.27.tgz
bosh upload-release garden-runc-release-1.9.5.tgz
bosh upload-release go-buildpack-release-1.8.11.tgz
bosh upload-release java-buildpack-release-4.6.tgz
bosh upload-release loggregator-release-99.tgz
bosh upload-release nats-release-22.tgz
bosh upload-release nodejs-buildpack-release-1.6.8.tgz
bosh upload-release php-buildpack-release-4.3.42.tgz
bosh upload-release python-buildpack-release-1.5.26.tgz
bosh upload-release ruby-buildpack-release-1.7.3.tgz
bosh upload-release staticfile-buildpack-release-1.4.16.tgz
bosh upload-release statsd-injector-release-1.0.30.tgz
bosh upload-release uaa-release-52.2.tgz
```

Modify `releases` section in `cf-deployment.yml` (important in China regions, skip it in oversea region)

```yaml
...
releases:
- name: binary-buildpack
  version: latest
- name: capi
  version: latest
- name: cf-mysql
  version: latest
- name: cf-networking
  version: latest
- name: cf-smoke-tests
  version: latest
- name: cf-syslog-drain
  version: latest
- name: cflinuxfs2
  version: latest
- name: consul
  version: latest
- name: diego
  version: latest
- name: dotnet-core-buildpack
  version: latest
- name: garden-runc
  version: latest
- name: go-buildpack
  version: latest
- name: java-buildpack
  version: latest
- name: loggregator
  version: latest
- name: nats
  version: latest
- name: nodejs-buildpack
  version: latest
- name: php-buildpack
  version: latest
- name: python-buildpack
  version: latest
- name: routing
  version: latest
- name: ruby-buildpack
  version: latest
- name: staticfile-buildpack
  version: latest
- name: statsd-injector
  version: latest
- name: uaa
  version: latest
```

Setup Domain, use your domain name

```
export CF_DOMAIN=...
```

Install Cloud Foundry

```
bosh -e my-bosh -d cf deploy cf-deployment.yml \
  --vars-store cf-vars.yml \
  -v system_domain=$CF_DOMAIN
```

Login Cloud Foundry

```
cf login -a http://api.$CF_DOMAIN --skip-ssl-validation -u admin -p `bosh int ./cf-vars.yml --path /cf_admin_password`
```

Update buildpacks (important in China regions, skip it in oversea region)
Default cf buildpacks need to download during `cf push`, but is unusable in China region, so download [offline-buildpacks](http://bosh-alicloud.oss-cn-hangzhou.aliyuncs.com/cf-offline-buildpacks_20171107) and update.

```
cf update-buildpack staticfile_buildpack -p staticfile_buildpack-cached-v1.4.18.zip -i 1
cf update-buildpack java_buildpack -p ~/Downloads/java-buildpack-offline-dad0000.zip -i 2
cf update-buildpack ruby_buildpack -p ruby_buildpack-cached-v1.7.5.zip -i 3
cf update-buildpack go_buildpack -p go_buildpack-cached-v1.8.13.zip -i 6
```

Enjoy your Cloud Foundry