// +build medium

/*
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2015 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package influxdb

import (
	"bytes"
	"encoding/gob"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/core/ctypes"

	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	//Do a ping to make sure the docker image actually came up. Otherwise this can fail Travis builds
	for i := 0; i < 3; i++ {
		resp, err := http.Get("http://" + os.Getenv("SNAP_INFLUXDB_HOST") + ":8086/ping")
		if err != nil || resp.StatusCode != 204 {
			//Try again after 3 seconds
			time.Sleep(3 * time.Second)
		} else {
			//Give the run.sh time to create the test database
			time.Sleep(5 * time.Second)
			return
		}
	}
	//If we got here, we failed to get to the server
	panic("Unable to connect to Influx host. Aborting test.")
}

func TestInfluxPublish(t *testing.T) {
	config := make(map[string]ctypes.ConfigValue)

	Convey("snap plugin InfluxDB integration testing with Influx", t, func() {
		var retention string

		if strings.HasPrefix(os.Getenv("INFLUXDB_VERSION"), "0.") {
			retention = "default"
		} else {
			retention = "autogen"
		}
		config["host"] = ctypes.ConfigValueStr{Value: os.Getenv("SNAP_INFLUXDB_HOST")}
		config["skip-verify"] = ctypes.ConfigValueBool{Value: false}
		config["user"] = ctypes.ConfigValueStr{Value: "root"}
		config["password"] = ctypes.ConfigValueStr{Value: "root"}
		config["database"] = ctypes.ConfigValueStr{Value: "test"}
		config["retention"] = ctypes.ConfigValueStr{Value: retention}
		config["debug"] = ctypes.ConfigValueBool{Value: false}
		config["log-level"] = ctypes.ConfigValueStr{Value: "debug"}
		config["precison"] = ctypes.ConfigValueStr{Value: "s"}

		tests("", config)

		config["scheme"] = ctypes.ConfigValueStr{Value: HTTP}
		config["port"] = ctypes.ConfigValueInt{Value: 8086}
		tests(HTTP, config)

		config["scheme"] = ctypes.ConfigValueStr{Value: UDP}
		config["port"] = ctypes.ConfigValueInt{Value: 4444}
		tests(UDP, config)
	})
}

func tests(scheme string, config map[string]ctypes.ConfigValue) {
	var buf bytes.Buffer

	ip := NewInfluxPublisher()
	cp, _ := ip.GetConfigPolicy()
	cfg, _ := cp.Get([]string{""}).Process(config)
	tags := map[string]string{"zone": "red"}

	Convey(" Publish integer metric via "+scheme, func() {
		metrics := []plugin.MetricType{
			*plugin.NewMetricType(core.NewNamespace("foo"), time.Now(), tags, "some unit", 99),
		}
		buf.Reset()
		enc := gob.NewEncoder(&buf)
		enc.Encode(metrics)
		err := ip.Publish(plugin.SnapGOBContentType, buf.Bytes(), *cfg)
		So(err, ShouldBeNil)
	})

	Convey("Publish float metric via "+scheme, func() {
		metrics := []plugin.MetricType{
			*plugin.NewMetricType(core.NewNamespace("bar"), time.Now(), tags, "some unit", 3.141),
		}
		buf.Reset()
		enc := gob.NewEncoder(&buf)
		enc.Encode(metrics)
		err := ip.Publish(plugin.SnapGOBContentType, buf.Bytes(), *cfg)
		So(err, ShouldBeNil)
	})

	Convey("Publish string metric via "+scheme, func() {
		metrics := []plugin.MetricType{
			*plugin.NewMetricType(core.NewNamespace("qux"), time.Now(), tags, "some unit", "bar"),
		}
		buf.Reset()
		enc := gob.NewEncoder(&buf)
		enc.Encode(metrics)
		err := ip.Publish(plugin.SnapGOBContentType, buf.Bytes(), *cfg)
		So(err, ShouldBeNil)
	})

	Convey("Publish boolean metric via "+scheme, func() {
		metrics := []plugin.MetricType{
			*plugin.NewMetricType(core.NewNamespace("baz"), time.Now(), tags, "some unit", true),
		}
		buf.Reset()
		enc := gob.NewEncoder(&buf)
		enc.Encode(metrics)
		err := ip.Publish(plugin.SnapGOBContentType, buf.Bytes(), *cfg)
		So(err, ShouldBeNil)
	})

	Convey("Publish multiple metrics via "+scheme, func() {
		metrics := []plugin.MetricType{
			*plugin.NewMetricType(core.NewNamespace("foo"), time.Now(), tags, "some unit", 101),
			*plugin.NewMetricType(core.NewNamespace("bar"), time.Now(), tags, "some unit", 5.789),
		}
		buf.Reset()
		enc := gob.NewEncoder(&buf)
		enc.Encode(metrics)
		err := ip.Publish(plugin.SnapGOBContentType, buf.Bytes(), *cfg)
		So(err, ShouldBeNil)
	})

	Convey("Publish dynamic metrics via "+scheme, func() {
		dynamicNS1 := core.NewNamespace("foo").
			AddDynamicElement("dynamic", "dynamic elem").
			AddStaticElement("bar")
		dynamicNS2 := core.NewNamespace("foo").
			AddDynamicElement("dynamic_one", "dynamic element one").
			AddDynamicElement("dynamic_two", "dynamic element two").
			AddStaticElement("baz")

		dynamicNS1[1].Value = "fooval"
		dynamicNS2[1].Value = "barval"
		dynamicNS2[2].Value = "bazval"

		metrics := []plugin.MetricType{
			*plugin.NewMetricType(dynamicNS1, time.Now(), tags, "", 123),
			*plugin.NewMetricType(dynamicNS2, time.Now(), tags, "", 456),
		}
		buf.Reset()
		enc := gob.NewEncoder(&buf)
		enc.Encode(metrics)
		err := ip.Publish(plugin.SnapGOBContentType, buf.Bytes(), *cfg)
		So(err, ShouldBeNil)
	})

	Convey("Publish nil value of metric via "+scheme, func() {
		metrics := []plugin.MetricType{
			*plugin.NewMetricType(core.NewNamespace("baz"), time.Now(), tags, "some unit", nil),
		}
		buf.Reset()
		enc := gob.NewEncoder(&buf)
		enc.Encode(metrics)
		err := ip.Publish(plugin.SnapGOBContentType, buf.Bytes(), *cfg)
		So(err, ShouldBeNil)
	})

	Convey("Publish multiple fields to one metric via "+scheme, func() {
		config["isMultiFields"] = ctypes.ConfigValueBool{Value: true}
		metrics := []plugin.MetricType{
			*plugin.NewMetricType(core.NewNamespace("a", "b", "x"), time.Now(), tags, "test unit", 123.6),
			*plugin.NewMetricType(core.NewNamespace("a", "b", "y"), time.Now(), tags, "test unit", 765.3),
			*plugin.NewMetricType(core.NewNamespace("a", "b", "z"), time.Now(), tags, "test unit", 12345),
			*plugin.NewMetricType(core.NewNamespace("a", "b", "z"), time.Now(), tags, "testunit", 11111),
		}

		buf.Reset()
		enc := gob.NewEncoder(&buf)
		enc.Encode(metrics)
		err := ip.Publish(plugin.SnapGOBContentType, buf.Bytes(), *cfg)
		So(err, ShouldBeNil)
	})

	Convey("Publish multiple fields to two metrics via "+scheme, func() {
		config["isMultiFields"] = ctypes.ConfigValueBool{Value: true}
		ntags := map[string]string{"zone": "red", "light": "yellow"}
		metrics := []plugin.MetricType{
			*plugin.NewMetricType(core.NewNamespace("influx", "x"), time.Now(), tags, "test unit", 333.6),
			*plugin.NewMetricType(core.NewNamespace("influx", "y"), time.Now(), tags, "test unit", 222.3),
			*plugin.NewMetricType(core.NewNamespace("influx", "z"), time.Now(), tags, "test unit", 1111),
			*plugin.NewMetricType(core.NewNamespace("influx", "r"), time.Now(), ntags, "unittest ", 777),
			*plugin.NewMetricType(core.NewNamespace("influx", "s"), time.Now(), ntags, "unittest", 888),
			*plugin.NewMetricType(core.NewNamespace("influx", "s"), time.Now(), ntags, "unit test", 999),
		}

		buf.Reset()
		enc := gob.NewEncoder(&buf)
		enc.Encode(metrics)
		err := ip.Publish(plugin.SnapGOBContentType, buf.Bytes(), *cfg)
		So(err, ShouldBeNil)
	})
}
