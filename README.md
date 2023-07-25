# scpw

![GitHub](https://img.shields.io/github/license/T-TRz879/scpw) ![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/T-TRz879/scpw)

scp client wrapper for automatic put or get.

![usage](./assets/scpw.gif)

## install

use `go get`

```
go get -u github.com/T-TRz879/scpw/cmd/scpw
```

## config

config file load in following order:

- `~/.scpw`
- `~/.scpw.yml`
- `~/.scpw.yaml`

config example:

<!-- prettier-ignore -->

```yaml
# To upload or download all the content in the directory(Exclude Root Directory), simply add the*
# (example: download /tmp -> /tmp/*)
- name: serverA
  user: appAdmin
  host: 10.0.16.18
  port: 22
  password: 123456
  type: PUT
  lr-map:
  # Put all content under /tmp to the remote /tmp/A directory
  - { local: /tmp/* , remote: /tmp/A/ }
  - { local: /tmp/a.txt , remote: /tmp/b.txt }

# Remote folders must end with /
- name: serverB
  user: appAdmin
  host: 10.0.16.17
  port: 22
  password: 123456
  type: GET
  lr-map:
  # Get all content from remote /root/lib to the local /home/appAdmin directory
  - { local: /home/appAdmin/ , remote: /root/lib/ }
  - { local: /home/appAdmin/redis.conf , remote: /root/redis.conf }

```
