package main

import (
	"bitbucket.org/evolutek/cellaserv2-protobuf"
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
)

func handleListServices(conn net.Conn, req *cellaserv.Request) {
	var servicesList []*ServiceJSON
	for _, names := range services {
		for _, s := range names {
			servicesList = append(servicesList, s.JSONStruct())
		}
	}

	data, err := json.Marshal(servicesList)
	if err != nil {
		log.Error("[Cellaserv] Could not marshal the services")
	}
	sendReply(conn, req, data)
}

func handleListConnections(conn net.Conn, req *cellaserv.Request) {
	var conns []string
	for c := range servicesConn {
		conns = append(conns, c.RemoteAddr().String())
	}
	data, err := json.Marshal(conns)
	if err != nil {
		log.Error("[Cellaserv] Could not marshal the connections")
	}
	sendReply(conn, req, data)
}

// handleGetLogs reply with the logs
func handleGetLogs(conn net.Conn, req *cellaserv.Request) {
	if req.Data == nil {
		log.Warning("[Cellaserv] Log request does not specify event")
		sendReplyError(conn, req, cellaserv.Reply_Error_BadArguments)
		return
	}

	event := string(req.Data)
	pattern := *logRootDirectory + "/" + event + ".log"
	filenames, err := filepath.Glob(pattern)

	if err != nil || len(filenames) == 0 {
		log.Warning("[Cellaserv] Log request specified erroneous log: ", err)
		sendReplyError(conn, req, cellaserv.Reply_Error_BadArguments)
		return
	}

	var data_buf bytes.Buffer

	for _, filename := range filenames {
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Warning("[Cellaserv] Could not open log:", filename)
			sendReplyError(conn, req, cellaserv.Reply_Error_BadArguments)
		}
		data_buf.Write(data)
	}
	sendReply(conn, req, data_buf.Bytes())
}

// handleShutdown quit cellaserv. Used for debug purposes
func handleShutdown() {
	stopProfiling()
	os.Exit(0)
}

func cellaservRequest(conn net.Conn, req *cellaserv.Request) {
	switch *req.Method {
	case "list-services":
		handleListServices(conn, req)
	case "list-connections":
		handleListConnections(conn, req)
	case "get-log":
		handleGetLogs(conn, req)
	case "shutdown":
		handleShutdown()
	default:
		sendReplyError(conn, req, cellaserv.Reply_Error_NoSuchMethod)
	}
}

func cellaservLog(pub *cellaserv.Publish) {
	if pub.Data == nil {
		log.Warning("[Log] %s does not have data", *pub.Event)
		return
	}
	data := string(pub.Data)
	event := (*pub.Event)[4:]
	logEvent(event, data)
}

func cellaservPublish(event string, data []byte) {
	pub := &cellaserv.Publish{Event: &event}
	if data != nil {
		pub.Data = data
	}
	pubBytes, err := proto.Marshal(pub)
	if err != nil {
		log.Error("[Cellaserv] Could not marshal event")
		return
	}
	msgType := cellaserv.Message_Publish
	msg := &cellaserv.Message{Type: &msgType, Content: pubBytes}
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		log.Error("[Cellaserv] Could not marshal event")
		return
	}

	doPublish(uint32(len(msgBytes)), msgBytes, pub)
}

// vim: set nowrap tw=100 noet sw=8:
