package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/NodePrime/jsonpath"
)

var cfgData []byte

func init() {
	cfgData, _ = ioutil.ReadFile("conf.json")
}

//Get config value
func Get(p string, value interface{}) error {
	paths, err := jsonpath.ParsePaths(p)
	if err == nil {
		eval, err := jsonpath.EvalPathsInBytes(cfgData, paths)
		if err == nil {
			if result, ok := eval.Next(); ok {
				return json.Unmarshal([]byte(result.Pretty(false)), &value)
			}
		}
		return err
	}
	return err
}
