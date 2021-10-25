# The `octane plugin` Manual

## Table of Contents
- [The `octane plugin` Manual](#the-octane-plugin-manual)
  - [Table of Contents](#table-of-contents)
  - [Using `octane`](#using-octane)
    - [Access to Octane](#access-to-octane)
    - [Usage of plugin](#usage-of-plugin)
## Using `Octane`

### Access to Octane
Reach out to ALM/Octane team in your org to get an workspace created for you (If not exists).
Get the following info:
- host_url : API endpoint to send the plax test results.
- client_id and client_secret : Request API access to get the keys
- workspace_id : ID of Workspace. (Can be retrieved from Swagger UI of Octane. Or Octane team can get you this info)
- shared_space_id : ID of Shared space. (Can be retrieved from Swagger UI of Octane. Or Octane team can get you this info)
- app_module_id : ID of Application Module. (Create an Application module in your Octane workspace.)

### Configuration
Add the following code to the plaxrun file. Paxrun identifies the plugin automatically and sends the result to Octane.

```
reports:
  octane:
    config:
      host_url: "https://almoctane-ams.saas.microfocus.com"
      client_id: "{OCTANE_CLIENT_ID}"
      client_secret: "{OCTANE_CLIENT_SECRET}"
      shared_space_id: "{SHARED_SPACE_ID}"
      workspace_id: "{WORKSPACE_ID}"
      app_module_id: 485024
      test_fields:     // optional
        Framework: plax
```

Notes:
- Data Types: app_module_id is a number and rest of the fields are strings
- Customize plax commands (if needed) to read the secrets from your secrets storage location.