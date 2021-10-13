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

- Export username and password as X_REPORT_PORTAL_USERNAME and X_REPORT_PORTAL_USERNAME in your code repo.

- Export report potal instance URL as RP_URL.
