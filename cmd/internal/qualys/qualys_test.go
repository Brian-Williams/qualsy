package qualys

import (
	"testing"
	"encoding/xml"
	"fmt"
	"reflect"
)

const (
	equalCriteriaF = `<?xml version="1.0" encoding="UTF-8"?>
<ServiceRequest>
  <filters>
    <Criteria field="name" operator="EQUALS">%s</Criteria>
  </filters>
</ServiceRequest>`
)

// Test for creating tag post search
func TestEqualBody(t *testing.T) {
	tag := "abc123"
	x := fmt.Sprintf(equalCriteriaF, tag)
	t.Logf("expected xml: %s", x)
	serviceRequest := equalBody(tag)

	t.Run("toStruct", func(t *testing.T) {
		var critWant CriteriaServiceRequest
		err := xml.Unmarshal([]byte(x), &critWant)
		if err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		if !reflect.DeepEqual(critWant, serviceRequest) {
			t.Errorf("expected:\n%+v\ngot:\n%+v", critWant, serviceRequest)
		}
	})

	t.Run("toXML", func(t *testing.T) {
		have, err := xmlString(serviceRequest)
		if err != nil {
			t.Fatalf("failed to unmarhsal: %s", err)
		}
		if x != have {
			t.Errorf("expected:\n%s\ngot:\n%+v", x, have)
		}
	})
}

func TestCreateTag(t *testing.T) {
	const (
		id = "25697744"
		colr = "#FFFFFF"
	)
	expected := fmt.Sprintf(xml.Header + `<ServiceRequest>
  <data>
    <Tag>
      <name>%s</name>
      <color>%s</color>
    </Tag>
  </data>
</ServiceRequest>`,
	id, colr)
	ct := CreateTag{
		Tag: TagInfo{
			id,
			colr,
		},
	}
	actual, err := xmlString(ct)
	if err != nil {
		t.Fatalf("failed to get string %+v: %s", ct, err)
	}
	if expected != actual {
		t.Log("expected: ", expected)
		t.Log("actual: ", actual)
		t.Errorf("CreateTag expected and actual don't match")
	}
}
