package mqtt

import "testing"

func TestDocs(t *testing.T) {
	(&MQTT{}).DocSpec().Write("mqtt")
}
