module github.com/Comcast/plax

go 1.14

replace github.com/Comcast/plax => ./

require (
	github.com/Comcast/sheens v0.9.1-0.20210115175817-a1a65cee59ac
	github.com/alecthomas/jsonschema v0.0.0-20210203201211-9145459e837c
	github.com/aws/aws-sdk-go v1.36.27
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dop251/goja v0.0.0-20210114204047-983fa61a23a8
	github.com/eclipse/paho.mqtt.golang v1.3.1
	github.com/go-sql-driver/mysql v1.6.0
	github.com/harlow/kinesis-consumer v0.3.4
	github.com/itchyny/gojq v0.12.2
	github.com/xeipuuv/gojsonschema v1.2.0
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	modernc.org/sqlite v1.10.6
)
