# `plaxrun`

A tool to run lots of Plax tests with various configurations.

See the [manual](../../doc/plaxrun.md) for more information.

# `Report Portal Plugin`

A tool that used for reporting test results.See the [Report Portal](https://reportportal.io/) for more details.

To enable this plugin in plaxrun report add below  details in plaxrun.yaml file and follow the steps.

```
reports:
  rp:
    dependsOn:
      - X_RP_TOKEN
    config:
      hostname: '{RP_URL}'
      token: "{X_RP_TOKEN}"
      project: 'Your Project Name'
```

- See the [README](https://github.comcast.com/xh-cloud/dht-tools/blob/main/README.md) to get the X_RP_TOKEN. 

- Export report potal instance URL as RP_URL.
