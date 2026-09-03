package handler

import (
	"bytes"
	"testing"

	corentity "GoCraft/core/entity"
)

func TestJavaAreaEffectCloudMetadataIncludesRadius(t *testing.T) {
	cloud := corentity.New(71, [16]byte{}, corentity.TypeAreaEffectCloud, 0, 64, 0)
	cloud.CloudRadius = 3
	cloud.AgeTicks = 4
	pkt := buildMobMetadata(cloud)
	if pkt == nil || pkt.ID != packetIDSetEntityData {
		t.Fatalf("cloud metadata packet = %+v", pkt)
	}
	radiusEntry := []byte{8, 3, 0x40, 0x40, 0, 0}
	waitingEntry := []byte{9, 8, 1}
	if !bytes.Contains(pkt.Data, radiusEntry) || !bytes.Contains(pkt.Data, waitingEntry) {
		t.Fatalf("cloud metadata omitted radius or warmup: %x", pkt.Data)
	}
}
