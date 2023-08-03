package basesuite

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Output struct {
	Name string
	Time time.Duration
}

func (s *BaseSuite) NewOutput(name string, err error, beginTime time.Time) {
	if err != nil {
		name = name + err.Error()
	}
	s.TestResults = append(s.TestResults, Output{Name: name, Time: time.Since(beginTime)})
}

func (s *BaseSuite) PrintTestResult() {
	var mutex sync.Mutex
	m := map[string]map[string]time.Duration{}
	go func() {
		for {
			mutex.Lock()
			outputs := s.TestResults
			s.TestResults = []Output{}
			mutex.Unlock()
			for _, output := range outputs {
				_, ok := m[output.Name]
				if ok {
					if m[output.Name]["maxTime"] < output.Time {
						m[output.Name]["maxTime"] = output.Time
					}
					if m[output.Name]["minTime"] > output.Time {
						m[output.Name]["minTime"] = output.Time
					}
					m[output.Name]["countTime"] = m[output.Name]["countTime"] + output.Time
					m[output.Name]["count"] = m[output.Name]["count"] + 1
					m[output.Name]["avgTime"] = m[output.Name]["countTime"] / m[output.Name]["count"]
				} else {
					m[output.Name] = map[string]time.Duration{
						"maxTime":   output.Time,
						"avgTime":   output.Time,
						"minTime":   output.Time,
						"countTime": output.Time,
						"count":     time.Duration(1),
					}
				}
			}
			var keys []string
			for k := range m {
				keys = append(keys, k)
			}
			// To store the keys in slice in sorted order
			sort.Strings(keys)

			fmt.Printf("\n%20s %20s %20s %20s %20s\n", "Name", "Count", "AvgTime", "MinTime", "MaxTime")
			for _, k := range keys {
				fmt.Printf("%25s  %20d %20vms %20vms %20vms\n", k, m[k]["count"], m[k]["avgTime"].Milliseconds(), m[k]["minTime"].Milliseconds(), m[k]["maxTime"].Milliseconds())
			}

			time.Sleep(10 * time.Second)
		}
	}()

}
