package utils

import (
	"errors"
	"reflect"
	"sync"
)

type ParametersIterator struct {
	sync.Mutex
	parameters []interface{}
	index      int
}

func NewParametersIterator(slice interface{}) ParametersIterator {
	val := reflect.ValueOf(slice)
	if val.Kind() != reflect.Slice {
		panic(errors.New("params not slice"))
	}
	sliceLen := val.Len()
	arr := make([]interface{}, sliceLen)
	for i := 0; i < sliceLen; i++ {
		arr[i] = val.Index(i).Interface()
	}
	return ParametersIterator{
		parameters: arr,
		index:      0,
	}
}
func (iter *ParametersIterator) Next() interface{} {
	iter.Lock()
	defer iter.Unlock()
	var selectedParameters interface{}
	if len(iter.parameters) == 0 {
		return nil
	} else if iter.index < len(iter.parameters) {
		selectedParameters = iter.parameters[iter.index]
	} else {
		index := iter.index % len(iter.parameters)
		selectedParameters = iter.parameters[index]
	}
	iter.index++
	return selectedParameters
}
func (iter *ParametersIterator) NextWithOutLoop() interface{} {
	iter.Lock()
	defer iter.Unlock()
	var selectedParameters interface{}
	if len(iter.parameters) == 0 {
		return nil
	} else if iter.index < len(iter.parameters) {
		selectedParameters = iter.parameters[iter.index]
	} else {
		return nil
	}
	iter.index++
	return selectedParameters
}
func (iter *ParametersIterator) Add(inter interface{}) {
	iter.Lock()
	defer iter.Unlock()
	iter.parameters = append(iter.parameters, inter)
}
func (iter *ParametersIterator) Len() int {
	return len(iter.parameters)
}
func (iter *ParametersIterator) GetParameters() []interface{} {
	return iter.parameters
}
