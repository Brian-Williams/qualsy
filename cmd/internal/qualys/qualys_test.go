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
		b, err := xml.MarshalIndent(serviceRequest, "", "  ")
		if err != nil {
			t.Fatalf("failed to unmarhsal: %s", err)
		}
		have := xml.Header + string(b)
		if x != have {
			t.Errorf("expected:\n%s\ngot:\n%+v", x, have)
		}
	})
}
