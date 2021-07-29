package internal

import (
	"bytes"
	"io/ioutil"
	"sync"
	"testing"
)

func TestMultiplexingReader_MultipleReaders(t *testing.T) {
	buffer := MultiplexingBuffer{}

	_, _ = buffer.Write([]byte("test"))

	_ = buffer.Close()

	reader1 := buffer.Reader()
	reader2 := buffer.Reader()

	bytes1, _ := ioutil.ReadAll(reader1)
	bytes2, _ := ioutil.ReadAll(reader2)

	if !bytes.Equal(bytes1, bytes2) {
		t.Fatal("Reader returned different content")
	}
}

func TestMultiplexingBuffer_StreamsData(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	buffer := MultiplexingBuffer{}

	reader := buffer.Reader()

	var read []byte

	go func() {
		read, _ = ioutil.ReadAll(reader)
		wg.Done()
	}()

	data := []byte("test")
	_, _ = buffer.Write(data)

	_ = buffer.Close()

	wg.Wait()
	if !bytes.Equal(data, read) {
		t.Fatal("Reader returned different content")
	}

}

func TestMultiplexingBuffer_StreamsBufferedData(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	buffer := MultiplexingBuffer{}

	data1 := []byte("test1")
	_, _ = buffer.Write(data1)

	reader := buffer.Reader()

	var read []byte

	go func() {
		read, _ = ioutil.ReadAll(reader)
		wg.Done()
	}()

	data2 := []byte("test2")
	_, _ = buffer.Write(data2)

	_ = buffer.Close()

	wg.Wait()
	if !bytes.Equal(append(data1, data2...), read) {
		t.Fatal("Reader returned different content")
	}

}
