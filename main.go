/*
Напишите код, реализующий пайплайн, работающий с целыми числами и состоящий из следующих стадий:

Стадия фильтрации отрицательных чисел (не пропускать отрицательные числа).
Стадия фильтрации чисел, не кратных 3 (не пропускать такие числа), исключая также и 0.
Стадия буферизации данных в кольцевом буфере с интерфейсом, соответствующим тому, который был дан в качестве задания в 19 модуле. В этой стадии предусмотреть опустошение буфера
(и соответственно, передачу этих данных, если они есть, дальше) с определённым интервалом во времени.
Значения размера буфера и этого интервала времени сделать настраиваемыми (как мы делали: через константы или глобальные переменные).
Написать источник данных для конвейера. Непосредственным источником данных должна быть консоль.

Также написать код потребителя данных конвейера. Данные от конвейера можно направить снова в консоль построчно, сопроводив их каким-нибудь поясняющим текстом,
например: «Получены данные …».

При написании источника данных подумайте о фильтрации нечисловых данных, которые можно ввести через консоль. Как и где их фильтровать, решайте сами.
*/

package main

import (
	"bufio"
	"container/ring"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Ring Buffer Cleaning Interval
const bufferDrainInterval time.Duration = 10 * time.Second

// Ring Buffer Size
const bufferSize int = 20

// Mess Ring Buffer Size
const bufferMessSize int = 100

// Ring Buffer Struct
type IntDataType struct {
	value    int
	resource string
}

// Ring Buffer Struct
type RingIntBuffer struct {
	array []IntDataType
	size  int
	head  int
	tail  int
	m     sync.Mutex
}

// NewRingIntBuffer - create a new integer buffer
func NewRingIntBuffer(size int) *RingIntBuffer {
	return &RingIntBuffer{
		array: make([]IntDataType, size),
		size:  size,
		head:  -1,
		tail:  -1,
	}
}

// Push adding a new element to the end of the buffer
func (r *RingIntBuffer) Push(el IntDataType) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.head == -1 {
		r.head = 0
		r.tail = 0
		r.array[r.head] = el
	} else if (r.tail+1)%r.size == r.head {
		r.head = (r.head + 1) % r.size
		r.tail = (r.tail + 1) % r.size
		r.array[r.tail] = el
	} else {
		r.tail = (r.tail + 1) % r.size
		r.array[r.tail] = el
	}
}

// Get - getting all the elements of the buffer and then clearing it
func (r *RingIntBuffer) Get() []IntDataType {
	if r.tail < 0 {
		return nil
	}
	r.m.Lock()
	defer r.m.Unlock()

	var output []IntDataType
	if r.head <= r.tail {
		output = r.array[r.head : r.tail+1]
	} else {
		output = append(r.array[r.head:], r.array[:r.tail+1]...)
	}

	r.head = -1
	r.tail = -1
	return output
}

// Mess Ring Buffer datatype
type RingMessStruct struct {
	mess      string
	inputData string
	datetime  time.Time
	success   bool
}

// Mess Ring Buffer
type RingMessBuffer struct {
	data  *ring.Ring
	mutex sync.Mutex
}

// create a new mess buffer
func NewRingMessBuffer(size int) *RingMessBuffer {
	return &RingMessBuffer{data: ring.New(size)}
}

// Push adding a new element to the end of the mess buffer
func (r *RingMessBuffer) Push(el RingMessStruct) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.data.Value = el
	r.data = r.data.Next()
}

// Get the length of the mess buffer
func (r *RingMessBuffer) Len() int {
	return r.data.Len()
}

// Iterate through the ring and print its contents
func (r *RingMessBuffer) GetStr() ([]string, int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	result := make([]string, r.Len())
	i := 0
	r.data.Do(func(x interface{}) {
		if x != nil {
			el := x.(RingMessStruct)
			result[i] = fmt.Sprintf("%v: %v := %v, success: %v", el.datetime.Format("2006-01-02 15:04:05.000"), el.mess, el.inputData, el.success)
			i++
		}
	})
	return result, i
}

// Clear the mess buffer
func (r *RingMessBuffer) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	len := r.Len()
	r.data = ring.New(len)
}

// The program accepts data from the console, leaving only integers.
func reportWorking(messBuffPushData <-chan RingMessStruct, messBuffGetDataCmd <-chan bool, messBuffGetData chan<- string, done chan bool, bufferMessSize int) {
	messBuff := NewRingMessBuffer(bufferMessSize)
	for {
		select {
		case data := <-messBuffPushData:
			messBuff.Push(data)
		case <-messBuffGetDataCmd:
			bufferData, len := messBuff.GetStr()
			if bufferData != nil {
				for i := 0; i < len; i++ {
					messBuffGetData <- bufferData[i]
				}
			}
		case <-done:
			return
		}
	}
}

// Simulating an additional data stream
func generator(min, max int, resource string, ch chan<- IntDataType, done chan bool, valueDrainInterval time.Duration, messBuffPushData chan<- RingMessStruct, messBuffGetDataCmd chan<- bool) {
	nameResource := resource
	ticker := time.NewTicker(valueDrainInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			value := rand.Intn(max-min+1) + min
			messBuffPushData <- RingMessStruct{mess: nameResource, inputData: fmt.Sprintf("%v", value), datetime: time.Now(), success: true}
			ch <- IntDataType{value, resource}
		}
	}
}

// Channel compaction function
func multiplexingFunc(done chan bool, channels ...chan IntDataType) <-chan IntDataType {
	dataToChan := make(chan IntDataType)
	go func() {
		var wg sync.WaitGroup
		multiplex := func(c <-chan IntDataType) {
			defer wg.Done()
			for {
				select {
				case i := <-c:
					dataToChan <- i
				case <-done:
					return
				}
			}
		}
		wg.Add(len(channels))
		for _, c := range channels {
			go multiplex(c)
		}
		go func() {
			wg.Wait()
		}()
	}()
	return dataToChan
}

// The program accepts data from the console, leaving only integers.
func readDataFromConsol(dataToChan chan<- IntDataType, done chan bool, messBuffPushData chan<- RingMessStruct, messBuffGetDataCmd chan<- bool) {
	scanner := bufio.NewScanner(os.Stdin)
	nameResource := "console"
	var value string
	fmt.Println("The program has started working: (<report> - for reporting, <exit> - for closing programm):")
	for scanner.Scan() {
		value = scanner.Text()
		switch strings.ToLower(value) {
		case "exit":
			fmt.Println("The program has completed its work! Bye!")
			close(done)
			return
		case "report":
			fmt.Println("Report:")
			messBuffGetDataCmd <- true
			continue
		}
		i, err := strconv.Atoi(value)
		if err != nil {
			fmt.Println("Sorry, but the program only processes integers.")
			messBuffPushData <- RingMessStruct{mess: nameResource, inputData: fmt.Sprintf("%v", value), datetime: time.Now(), success: false}
			continue
		}
		messBuffPushData <- RingMessStruct{mess: nameResource, inputData: fmt.Sprintf("%v", value), datetime: time.Now(), success: true}
		dataToChan <- IntDataType{i, nameResource}

	}
}

// Pipeline stage that processes integers
type StageIntDataType func(<-chan IntDataType, <-chan bool, chan<- RingMessStruct) <-chan IntDataType

// Integer Processing Pipeline
type PipeLineInt struct {
	stages       []StageIntDataType
	done         <-chan bool
	ringBuffMess chan<- RingMessStruct
}

// Creating an integer processing pipeline
func NewPipelineInt(done <-chan bool, messBuffPushData chan<- RingMessStruct, stages ...StageIntDataType) *PipeLineInt {
	return &PipeLineInt{done: done, stages: stages, ringBuffMess: messBuffPushData}
}

// Running the integer processing pipeline
func (p *PipeLineInt) Run(source <-chan IntDataType) <-chan IntDataType {
	var c <-chan IntDataType = source
	for index := range p.stages {
		c = p.runStageIntDataType(p.stages[index], c)
	}
	return c
}

// Start of a separate stage of the pipeline
func (p *PipeLineInt) runStageIntDataType(stage StageIntDataType, sourceChan <-chan IntDataType) <-chan IntDataType {
	return stage(sourceChan, p.done, p.ringBuffMess)
}

// 1-stage. Negative number filtering stage (do not pass negative numbers).
func funcFilterStage1(prevData <-chan IntDataType, done <-chan bool, messBuffPushData chan<- RingMessStruct) <-chan IntDataType {
	convertedIntChan := make(chan IntDataType)
	nameResource := "stage1"
	go func() {
		for {
			select {
			case data := <-prevData:
				messSuccess := false
				if data.value > 0 {
					messSuccess = true
					select {
					case convertedIntChan <- data:
					case <-done:
						return
					}
					messBuffPushData <- RingMessStruct{mess: fmt.Sprintf("%v.%v", data.resource, nameResource), inputData: fmt.Sprintf("%v", data.value), datetime: time.Now(), success: messSuccess}
				}
			case <-done:
				return
			}
		}
	}()
	return convertedIntChan
}

// 2-stage. The stage of filtering numbers that are not multiples of 3 (do not skip such numbers), also excluding 0.
func funcFilterStage2(prevData <-chan IntDataType, done <-chan bool, messBuffPushData chan<- RingMessStruct) <-chan IntDataType {
	convertedIntChan := make(chan IntDataType)
	nameResource := "stage2"
	go func() {
		for {
			select {
			case data := <-prevData:
				messSuccess := false
				if data.value != 0 && data.value%3 == 0 {
					messSuccess = true
					select {
					case convertedIntChan <- data:
					case <-done:
						return
					}
					messBuffPushData <- RingMessStruct{mess: fmt.Sprintf("%v.%v", data.resource, nameResource), inputData: fmt.Sprintf("%v", data.value), datetime: time.Now(), success: messSuccess}
				}
			case <-done:
				return
			}
		}
	}()
	return convertedIntChan
}

// 3-stage. Stage of buffering data in a ring buffer with an interface.
func funcFilterStage3(prevData <-chan IntDataType, done <-chan bool, messBuffPushData chan<- RingMessStruct) <-chan IntDataType {
	convertedIntChan := make(chan IntDataType)
	nameResource := "stage3"
	bufferRing := NewRingIntBuffer(bufferSize)
	go func() {
		for {
			select {
			case data := <-prevData:
				bufferRing.Push(data)
				messBuffPushData <- RingMessStruct{mess: fmt.Sprintf("%v.%v insert bufer", data.resource, nameResource), inputData: fmt.Sprintf("%v", data.value), datetime: time.Now(), success: true}
			case <-done:
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-time.After(bufferDrainInterval):
				bufferData := bufferRing.Get()
				if bufferData != nil {
					for _, data := range bufferData {
						select {
						case convertedIntChan <- data:
						case <-done:
							return
						}
					}
					messBuffPushData <- RingMessStruct{mess: fmt.Sprintf("get data from buffer [%v]", bufferData), datetime: time.Now(), success: true}
				}
			case <-done:
				return
			}
		}
	}()
	return convertedIntChan
}

func main() {
	done := make(chan bool)

	messBuffPushData := make(chan RingMessStruct)
	messBuffGetDataCmd := make(chan bool)
	messBuffGetData := make(chan string)

	// The program accepts data from the console, leaving only integers.
	go reportWorking(messBuffPushData, messBuffGetDataCmd, messBuffGetData, done, bufferMessSize)

	// The program accepts data from the console, leaving only integers.
	inputDataFromConsole := make(chan IntDataType)
	go readDataFromConsol(inputDataFromConsole, done, messBuffPushData, messBuffGetDataCmd)
	// Simulating an additional data stream
	inputDataFromChannel1 := make(chan IntDataType)
	go generator(100, 199, "Resource1", inputDataFromChannel1, done, time.Duration(10*time.Second), messBuffPushData, messBuffGetDataCmd)
	inputDataFromChannel2 := make(chan IntDataType)
	go generator(200, 299, "Resource2", inputDataFromChannel2, done, time.Duration(15*time.Second), messBuffPushData, messBuffGetDataCmd)
	inputDataFromChannel3 := make(chan IntDataType)
	go generator(300, 399, "Resource3", inputDataFromChannel3, done, time.Duration(20*time.Second), messBuffPushData, messBuffGetDataCmd)

	// Pipeline data consumer
	consumer := func(done <-chan bool, c <-chan IntDataType) {
		for {
			select {
			case data := <-c:
				fmt.Printf("%v: Data received from buffer: %v %v\n", time.Now().Format("2006-01-02 15:04:05.000"), data.value, data.resource)
			case dataMess := <-messBuffGetData:
				fmt.Println(dataMess)
			case <-done:
				return
			}
		}
	}

	source := multiplexingFunc(done, inputDataFromConsole, inputDataFromChannel1, inputDataFromChannel2, inputDataFromChannel3)
	pipeline := NewPipelineInt(done, messBuffPushData, funcFilterStage1, funcFilterStage2, funcFilterStage3)
	consumer(done, pipeline.Run(source))
}
