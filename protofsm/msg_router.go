package protofsm

// For some reason golangci-lint has a false positive on the sort order of the
// imports for the new "maps" package... We need the nolint directive here to
// ignore that.
//
//nolint:gci
import (
	"fmt"
	"maps"
	"sync"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/lightningnetwork/lnd/fn"
	"github.com/lightningnetwork/lnd/lnwire"
)

var (
	// ErrDuplicateEndpoint is returned when an endpoint is registered with
	// a name that already exists.
	ErrDuplicateEndpoint = fmt.Errorf("endpoint already registered")

	// ErrUnableToRouteMsg is returned when a message is unable to be
	// routed to any endpoints.
	ErrUnableToRouteMsg = fmt.Errorf("unable to route message")
)

// EndpointName is the name of a given endpoint. This MUST be unique across all
// registered endpoints.
type EndpointName = string

// PeerMsg is a wire message that includes the public key of the peer that sent
// it.
type PeerMsg struct {
	lnwire.Message

	// PeerPub is the public key of the peer that sent this message.
	PeerPub btcec.PublicKey
}

// MsgEndpoint is an interface that represents a message endpoint, or the
// sub-system that will handle processing an incoming wire message.
type MsgEndpoint interface {
	// Name returns the name of this endpoint. This MUST be unique across
	// all registered endpoints.
	Name() EndpointName

	// CanHandle returns true if the target message can be routed to this
	// endpoint.
	CanHandle(msg PeerMsg) bool

	// SendMessage handles the target message, and returns true if the
	// message was able being processed.
	SendMessage(msg PeerMsg) bool
}

// MsgRouter is an interface that represents a message router, which is generic
// sub-system capable of routing any incoming wire message to a set of
// registered endpoints.
type MsgRouter interface {
	// RegisterEndpoint registers a new endpoint with the router. If a
	// duplicate endpoint exists, an error is returned.
	RegisterEndpoint(MsgEndpoint) error

	// UnregisterEndpoint unregisters the target endpoint from the router.
	UnregisterEndpoint(EndpointName) error

	// RouteMsg attempts to route the target message to a registered
	// endpoint. If ANY endpoint could handle the message, then nil is
	// returned. Otherwise, ErrUnableToRouteMsg is returned.
	RouteMsg(PeerMsg) error

	// Start starts the peer message router.
	Start()

	// Stop stops the peer message router.
	Stop()
}

// queryMsg is a message sent into the main event loop to query or modify the
// internal state.
type queryMsg[Q any, R any] struct {
	query Q

	respChan chan fn.Either[R, error]
}

// SendError sends a response to the query with the given error value.
func (q *queryMsg[Q, R]) SendError(err error) {
	q.respChan <- fn.NewRight[R, error](err)
}

// SendResponse sends a response to the query with the given response value.
func (q *queryMsg[Q, R]) SendResponse(resp R) {
	q.respChan <- fn.NewLeft[R, error](resp)
}

// sendQuery sends a query to the main event loop, and returns the response.
func sendQuery[Q any, R any](sendChan chan queryMsg[Q, R], queryArg Q,
	quit chan struct{}) fn.Either[R, error] {

	query := queryMsg[Q, R]{
		query:    queryArg,
		respChan: make(chan fn.Either[R, error], 1),
	}

	if !fn.SendOrQuit(sendChan, query, quit) {
		return fn.NewRight[R](fmt.Errorf("router shutting down"))
	}

	resp, err := fn.RecvResp(query.respChan, nil, quit)
	if err != nil {
		return fn.NewRight[R](err)
	}

	return resp
}

// sendQueryErr is a helper function based on sendQuery that can be used when
// the query only needs an error response.
func sendQueryErr[Q any](sendChan chan queryMsg[Q, error], queryArg Q,
	quitChan chan struct{}) error {

	var err error
	resp := sendQuery(sendChan, queryArg, quitChan)
	resp.WhenRight(func(e error) {
		err = e
	})
	resp.WhenLeft(func(e error) {
		err = e
	})

	return err
}

// EndpointsMap is a map of all registered endpoints.
type EndpointsMap map[EndpointName]MsgEndpoint

// MultiMsgRouter is a type of message router that is capable of routing new
// incoming messages, permitting a message to be routed to multiple registered
// endpoints.
type MultiMsgRouter struct {
	startOnce sync.Once
	stopOnce  sync.Once

	// registerChan is the channel that all new endpoints will be sent to.
	registerChan chan queryMsg[MsgEndpoint, error]

	// unregisterChan is the channel that all endpoints that are to be
	// removed are sent to.
	unregisterChan chan queryMsg[EndpointName, error]

	// msgChan is the channel that all messages will be sent to for
	// processing.
	msgChan chan queryMsg[PeerMsg, error]

	// endpointsQueries is a channel that all queries to the endpoints map
	// will be sent to.
	endpointQueries chan queryMsg[MsgEndpoint, EndpointsMap]

	wg   sync.WaitGroup
	quit chan struct{}
}

// NewMultiMsgRouter creates a new instance of a peer message router.
func NewMultiMsgRouter() *MultiMsgRouter {
	return &MultiMsgRouter{
		registerChan:    make(chan queryMsg[MsgEndpoint, error]),
		unregisterChan:  make(chan queryMsg[EndpointName, error]),
		msgChan:         make(chan queryMsg[PeerMsg, error]),
		endpointQueries: make(chan queryMsg[MsgEndpoint, EndpointsMap]),
		quit:            make(chan struct{}),
	}
}

// Start starts the peer message router.
func (p *MultiMsgRouter) Start() {
	log.Infof("Starting MsgRouter")

	p.startOnce.Do(func() {
		p.wg.Add(1)
		go p.msgRouter()
	})
}

// Stop stops the peer message router.
func (p *MultiMsgRouter) Stop() {
	log.Infof("Stopping MsgRouter")

	p.stopOnce.Do(func() {
		close(p.quit)
		p.wg.Wait()
	})
}

// RegisterEndpoint registers a new endpoint with the router. If a duplicate
// endpoint exists, an error is returned.
func (p *MultiMsgRouter) RegisterEndpoint(endpoint MsgEndpoint) error {
	return sendQueryErr(p.registerChan, endpoint, p.quit)
}

// UnregisterEndpoint unregisters the target endpoint from the router.
func (p *MultiMsgRouter) UnregisterEndpoint(name EndpointName) error {
	return sendQueryErr(p.unregisterChan, name, p.quit)
}

// RouteMsg attempts to route the target message to a registered endpoint. If
// ANY endpoint could handle the message, then true is
// returned.
func (p *MultiMsgRouter) RouteMsg(msg PeerMsg) error {
	return sendQueryErr(p.msgChan, msg, p.quit)
}

// Endpoints returns a list of all registered endpoints.
func (p *MultiMsgRouter) Endpoints() EndpointsMap {
	resp := sendQuery(p.endpointQueries, nil, p.quit)

	var endpoints EndpointsMap
	resp.WhenLeft(func(e EndpointsMap) {
		endpoints = e
	})

	return endpoints
}

// msgRouter is the main goroutine that handles all incoming messages.
func (p *MultiMsgRouter) msgRouter() {
	defer p.wg.Done()

	// endpoints is a map of all registered endpoints.
	endpoints := make(map[EndpointName]MsgEndpoint)

	for {
		select {
		// A new endpoint was just sent in, so we'll add it to our set
		// of registered endpoints.
		case newEndpointMsg := <-p.registerChan:
			endpoint := newEndpointMsg.query

			log.Infof("MsgRouter: registering new MsgEndpoint(%s)",
				endpoint.Name())

			// If this endpoint already exists, then we'll return
			// an error as we require unique names.
			if _, ok := endpoints[endpoint.Name()]; ok {
				log.Errorf("MsgRouter: rejecting duplicate "+
					"endpoint: %v", endpoint.Name())

				newEndpointMsg.SendError(ErrDuplicateEndpoint)

				continue
			}

			endpoints[endpoint.Name()] = endpoint

			newEndpointMsg.SendError(nil)

		// A request to unregister an endpoint was just sent in, so
		// we'll attempt to remove it.
		case endpointName := <-p.unregisterChan:
			delete(endpoints, endpointName.query)

			log.Infof("MsgRouter: unregistering MsgEndpoint(%s)",
				endpointName.query)

			endpointName.SendError(nil)

		// A new message was just sent in. We'll attempt to route it to
		// all the endpoints that can handle it.
		case msgQuery := <-p.msgChan:
			msg := msgQuery.query

			// Loop through all the endpoints and send the message
			// to those that can handle it the message.
			var couldSend bool
			for _, endpoint := range endpoints {
				if endpoint.CanHandle(msg) {
					log.Debugf("MsgRouter: sending msg %T "+
						"to endpoint %s", msg.Message,
						endpoint.Name())

					sent := endpoint.SendMessage(msg)
					couldSend = couldSend || sent
				}
			}

			var err error
			if !couldSend {
				log.Debugf("MsgRouter: unable to route msg %T",
					msg)

				err = ErrUnableToRouteMsg
			}

			msgQuery.SendError(err)

		// A query for the endpoint state just came in, we'll send back
		// a copy of our current state.
		case endpointQuery := <-p.endpointQueries:
			endpointsCopy := make(EndpointsMap, len(endpoints))
			maps.Copy(endpointsCopy, endpoints)

			endpointQuery.SendResponse(endpointsCopy)

		case <-p.quit:
			return
		}
	}
}

// A compile time check to ensure MultiMsgRouter implements the MsgRouter
// interface.
var _ MsgRouter = (*MultiMsgRouter)(nil)
