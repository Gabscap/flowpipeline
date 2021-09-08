// Captures Netflow v9 and feeds flows to the following segments. Currently,
// this segment only uses a limited subset of goflow2 functionality.
// If no configuration option is provided a sflow and a netflow collector will be started.
// netflowLagcy is also built in but currently not tested.
package goflow

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/bwNetFlow/flowpipeline/segments"
	flow "github.com/bwNetFlow/protobuf/go"
	"google.golang.org/protobuf/proto"

	"github.com/netsampler/goflow2/transport"
	"github.com/netsampler/goflow2/utils"
)

type Goflow struct {
	segments.BaseSegment
	ListenAddresses string // optional, default is "sflow://:6343,netflow://:2055"

	goflow_in chan *flow.FlowMessage
}

func (segment Goflow) New(config map[string]string) segments.Segment {

	var listenAddresses = "sflow://:6343,netflow://:2055"
	if config["listenAddresses"] != "" {
		listenAddresses = config["listenAddresses"]
		log.Printf("[info] Goflow: starting listeners for %s", listenAddresses)
	}

	return &Goflow{
		ListenAddresses: listenAddresses,
	}
}

func (segment *Goflow) Run(wg *sync.WaitGroup) {
	defer func() {
		close(segment.Out)
		wg.Done()
	}()
	segment.goflow_in = make(chan *flow.FlowMessage)
	go segment.startGoFlow(&channelDriver{segment.goflow_in})
	for {
		select {
		case msg, ok := <-segment.goflow_in:
			if !ok {
				return
			}
			segment.Out <- msg
		case msg, ok := <-segment.In:
			if !ok {
				return
			}
			segment.Out <- msg
		}
	}
}

type channelDriver struct {
	out chan *flow.FlowMessage
}

func (d *channelDriver) Send(key, data []byte) error {
	msg := &flow.FlowMessage{}
	// TODO: can we shave of this Unmarshal here and the Marshal in line 95
	if err := proto.Unmarshal(data, msg); err != nil {
		log.Println("[error] Goflow: Conversion error for received flow.")
		return nil
	}
	d.out <- msg
	return nil
}

func (d *channelDriver) Close(context.Context) error {
	close(d.out)
	return nil
}

type myProtobufDriver struct {
}

func (d *myProtobufDriver) Format(data interface{}) ([]byte, []byte, error) {
	msg, ok := data.(proto.Message)
	if !ok {
		return nil, nil, fmt.Errorf("message is not protobuf")
	}
	// TODO: can we shave of this Marshal here and the Unmarshal in line 72
	b, err := proto.Marshal(msg)
	return nil, b, err
}

func (d *myProtobufDriver) Prepare() error             { return nil }
func (d *myProtobufDriver) Init(context.Context) error { return nil }

func (segment *Goflow) startGoFlow(transport transport.TransportInterface) {
	wg := &sync.WaitGroup{}
	formatter := &myProtobufDriver{}

	for _, listenAddress := range strings.Split(segment.ListenAddresses, ",") {
		wg.Add(1)
		go func(listenAddress string) {
			defer wg.Done()
			listenAddrUrl, err := url.Parse(listenAddress)
			if err != nil {
				log.Fatal(err)
			}

			hostname := listenAddrUrl.Hostname()
			port, err := strconv.ParseUint(listenAddrUrl.Port(), 10, 64)
			if err != nil {
				log.Printf("[error] Goflow: Port %s could not be converted to integer", listenAddrUrl.Port())
				return
			}
			switch scheme := listenAddrUrl.Scheme; scheme {
			case "netflow":
				sNF := &utils.StateNetFlow{
					Format:    formatter,
					Transport: transport,
				}
				log.Printf("[info] Goflow: Listening for Netflow v9 on port %d...", port)
				err = sNF.FlowRoutine(1, hostname, int(port), false)
			case "sflow":
				sSFlow := &utils.StateSFlow{
					Format:    formatter,
					Transport: transport,
				}
				log.Printf("[info] Goflow: Listening for sflow on port %d...", port)
				err = sSFlow.FlowRoutine(1, hostname, int(port), false)
			case "nfl":
				sNFL := &utils.StateNFLegacy{
					Format:    formatter,
					Transport: transport,
				}
				log.Printf("[info] Goflow: Listening for sflow on port %d...", port)
				err = sNFL.FlowRoutine(1, hostname, int(port), false)
			default:
				log.Printf("[error] Goflow: scheme %s does not exist", listenAddrUrl.Scheme)
				return
			}
			if err != nil {
				log.Fatal(err)
			}

		}(listenAddress)
	}
	wg.Wait()
}

func init() {
	segment := &Goflow{}
	segments.RegisterSegment("goflow", segment)
}
