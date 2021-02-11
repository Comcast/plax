package chans

import "testing"

func TestDocsMQTT(t *testing.T) {
	(&MQTT{}).DocSpec().Write("mqtt")
}
