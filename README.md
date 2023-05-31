# scpw

scp client wrapper for automatic put or get.

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
# To upload or download all the content in the directory, simply add the*
# (example: download /tmp -> /tmp/*)
- name: serverA
  user: appAdmin
  host: 10.0.16.18
  port: 22
  password: 123456
  type: PUT
  lr-map:
  - { local: /tmp/* , remote: /tmp/A/ }
  - { local: /tmp/a.txt , remote: /tmp/b.txt }


- name: serverB
  user: appAdmin
  host: 10.0.16.17
  port: 22
  password: 123456
  type: GET
  lr-map:
  - { local: /home/appAdmin/lib/* , remote: /root/lib/ }
  - { local: /home/appAdmin/redis.conf , remote: /root/redis.conf }

```
