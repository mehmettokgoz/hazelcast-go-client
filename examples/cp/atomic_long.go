package main

import (
	"context"
	"fmt"
	"github.com/hazelcast/hazelcast-go-client"
	"log"
)

func logError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func checkAndPrint(value interface{}, err error) {
	logError(err)
	fmt.Println(value)
}

/*
	In order to use CP Subsystem in, you need to have at least three member and CP should be enabled in the XML.
	Zero member means CP Subsystem is disabled.
	<cp-subsystem>
		<cp-member-count>3</cp-member-count>
		<group-size>3</group-size>
	</cp-subsystem>
*/

func main() {
	ctx := context.Background()
	client, err := hazelcast.StartNewClient(ctx)
	logError(err)
	cp := client.CPSubsystem()
	viewCounter, err := cp.GetAtomicLong(context.Background(), "views")
	logError(err)
	val, err := viewCounter.Get(ctx)
	checkAndPrint(val, err)
	err = viewCounter.Set(ctx, 10)
	logError(err)
	val, err = viewCounter.Get(ctx)
	checkAndPrint(val, err)
	val, err = viewCounter.AddAndGet(ctx, 50)
	checkAndPrint(val, err)
	val, err = viewCounter.IncrementAndGet(ctx)
	checkAndPrint(val, err)
	r, err := viewCounter.CompareAndSet(ctx, 61, 62)
	checkAndPrint(r, err)
	val, err = viewCounter.DecrementAndGet(ctx)
	checkAndPrint(val, err)
}
