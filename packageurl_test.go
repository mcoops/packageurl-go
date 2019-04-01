/*
Copyright (c) the purl authors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package packageurl_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/package-url/packageurl-go"
)

type TestFixture struct {
	Description   string     `json:"description"`
	Purl          string     `json:"purl"`
	CanonicalPurl string     `json:"canonical_purl"`
	PackageType   string     `json:"type"`
	Namespace     string     `json:"namespace"`
	Name          string     `json:"name"`
	Version       string     `json:"version"`
	QualifierMap  OrderedMap `json:"qualifiers"`
	Subpath       string     `json:"subpath"`
	IsInvalid     bool       `json:"is_invalid"`
}

// OrderedMap is used to store the TestFixture.QualifierMap, to ensure that the
// declaration order of qualifiers is preserved.
type OrderedMap struct {
	OrderedKeys []string
	Map         map[string]string
}

// qualifiersMapPattern is used to parse the TestFixture "qualifiers" field to
// ensure that it's a json object.
var qualifiersMapPattern = regexp.MustCompile(`^\{.*\}$`)

// UnmarshalJSON unmarshals the qualifiers field for a TestFixture. The
// qualifiers field is given as a json object such as:
//
//        "qualifiers": {"arch": "i386", "distro": "fedora-25"}
//
// This function performs in-order parsing of these values into an OrderedMap to
// preserve items in order of declaration. Note that parsing as a
// map[string]string won't preserve element order.
func (m *OrderedMap) UnmarshalJSON(bytes []byte) error {
	data := string(bytes)
	switch data {
	case "null":
		m.OrderedKeys = []string{}
		m.Map = make(map[string]string, 0)
		return nil
	default:
		// ensure that the data is a json object "{...}"
		if !qualifiersMapPattern.MatchString(data) {
			return fmt.Errorf("qualifiers parse error: not a json object: %s", data)
		}

		// find out the order in which map keys occur
		dec := json.NewDecoder(strings.NewReader(data))
		// consume opening '{'
		_, _ = dec.Token()
		for dec.More() {
			t, _ := dec.Token()
			switch token := t.(type) {
			case json.Delim:
				if token != '}' {
					return fmt.Errorf("qualifiers parse error: expected delimiter '}', got: %v", token)
				}
				// closed json object -> we're done
				break
			case string:
				// this token is a dictionary key
				m.OrderedKeys = append(m.OrderedKeys, token)
				// consume the value (the token following the colon after the key)
				_, _ = dec.Token()
			}
		}

		// now that we know the key order, just fill the OrderedMap.Map field
		if err := json.Unmarshal(bytes, &m.Map); err != nil {
			return err
		}
		return nil
	}
}

// Qualifiers converts the TestFixture.QualifierMap field to an object of type
// packageurl.Qualifiers.
func (t TestFixture) Qualifiers() packageurl.Qualifiers {
	q := packageurl.Qualifiers{}

	for _, key := range t.QualifierMap.OrderedKeys {
		q = append(q, packageurl.Qualifier{Key: key, Value: t.QualifierMap.Map[key]})
	}

	return q
}

// TestFromStringExamples verifies that parsing example strings produce expected
// results.
func TestFromStringExamples(t *testing.T) {
	// Read the json file
	data, err := ioutil.ReadFile("testdata/test-suite-data.json")
	if err != nil {
		t.Fatal(err)
	}
	// Load the json file contents into a structure
	testData := []TestFixture{}
	err = json.Unmarshal(data, &testData)
	if err != nil {
		t.Fatal(err)
	}

	// Use FromString on each item in the test set
	for _, tc := range testData {
		// Should parse without issue
		result, err := packageurl.FromString(tc.Purl)
		if tc.IsInvalid == false {
			if err != nil {
				t.Logf("%s failed: %s", tc.Description, err)
				t.Fail()
			}
		} else {
			// Invalid cases
			if err == nil {
				t.Logf("%s did not fail and returned %#v", tc.Description, result)
				t.Fail()
			}
		}
	}
}

// TestToStringExamples verifies that the resulting package urls created match
// the expected format.
func TestToStringExamples(t *testing.T) {
	// Read the json file
	data, err := ioutil.ReadFile("testdata/test-suite-data.json")
	if err != nil {
		t.Fatal(err)
	}
	// Load the json file contents into a structure
	testData := []TestFixture{}
	err = json.Unmarshal(data, &testData)
	if err != nil {
		t.Fatal(err)
	}
	// Use ToString on each item
	for _, tc := range testData {
		// Skip invalid items
		if tc.IsInvalid == true {
			continue
		}
		instance := packageurl.NewPackageURL(
			tc.PackageType, tc.Namespace, tc.Name,
			tc.Version, tc.Qualifiers(), tc.Subpath)
		result := instance.ToString()

		// NOTE: We create a purl with ToString and then load into a PackageURL
		//       because qualifiers may not be in any order. By reparsing back
		//       we can ensure the data transfers between string and instance form.
		canonical, _ := packageurl.FromString(tc.CanonicalPurl)
		toTest, _ := packageurl.FromString(result)
		// If the two results don't equal then the ToString failed
		if !reflect.DeepEqual(toTest, canonical) {
			t.Logf("%s failed: %s != %s", tc.Description, result, tc.CanonicalPurl)
			t.Fail()
		}
	}
}

// TestStringer verifies that the Stringer implementation produces results
// equivalent with the ToString method.
func TestStringer(t *testing.T) {
	// Read the json file
	data, err := ioutil.ReadFile("testdata/test-suite-data.json")
	if err != nil {
		t.Fatal(err)
	}
	// Load the json file contents into a structure
	testData := []TestFixture{}
	err = json.Unmarshal(data, &testData)
	if err != nil {
		t.Fatal(err)
	}
	// Use ToString on each item
	for _, tc := range testData {
		// Skip invalid items
		if tc.IsInvalid == true {
			continue
		}
		instance := packageurl.NewPackageURL(
			tc.PackageType, tc.Namespace, tc.Name,
			tc.Version, tc.Qualifiers(), tc.Subpath)

		// Verify that the Stringer implementation returns a result
		// equivalent to ToString().
		if instance.ToString() != instance.String() {
			t.Logf("%s failed: Stringer implementation differs from ToString: %s != %s", tc.Description, instance.String(), instance.ToString())
			t.Fail()
		}
	}
}
