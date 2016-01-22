/*
Usage go run fortune-server.go [fserver RPC ip:port] [fserver UDP ip:port] [fortune-string]
*/

package main

import (
	//"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	//"strconv"
	"math/rand"
	"time"
	"net/rpc"
)

/////////// Error messages
type ErrMessage struct {
	Error string
}


//~~~ Message with details for contacting the fortune-server.
type FortuneInfoMessage struct {
	FortuneServer string
	FortuneNonce  int64
}

////////// Messages to/from fserver
// Response from the fortune-server containing the fortune.
type FortuneMessage struct {
	Fortune string
}

type FortuneReqMessage struct {
	FortuneNonce int64
}


////////// list of clients and their info
type Client struct{
	clientAddr string
	FortuneNonce int64
	nextClient *Client
}

type FortuneServerRPC struct{}


var fserverUDPAddr *net.UDPAddr
var fserverUDPString string
var fserverRPCAddr string
var fortuneString string
var clientList *Client

///Main Method
func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	clientList= nil

	//process args
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr,
			"Usage: %s [aserver UDP ip:port] [fserver RPC ip:port] [secret]\n",
			os.Args[0])
		os.Exit(1)
	}

	fserverRPCAddr = os.Args[1]
	

	fserverUDPString = os.Args[2]
	fserverUDPAddr, err := net.ResolveUDPAddr("udp", fserverUDPString)
	checkError(err)
	

	fortuneString = os.Args[3]

	//start go routine to listen for RPC calls
	go startRPCForClient()
	
	//listen for connection
	conn, _ := net.ListenUDP("udp", fserverUDPAddr)
	
	for{
		buf := make([]byte, 1024)
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
        		checkError(err)
        		os.Exit(1)
    		}
		
		verifyClient(remoteAddr.String(), buf[:n], conn)
		
		
		
	}
}

//
func sendErrorBack(status string, conn *net.UDPConn, clientToHandle *Client) {
	errMsg := ErrMessage{status}
	outBuffer, err := json.Marshal(errMsg)
	checkError(err)
	
	toClient, err := net.ResolveUDPAddr("udp", clientToHandle.clientAddr)
	checkError(err)
	
	_, err = conn.WriteTo(outBuffer, toClient)
	checkError(err)
}

//
func verifyClient(clientAddr string, msg []byte, conn *net.UDPConn) string {
	clientToHandle := clientList
	if (clientToHandle.clientAddr == clientAddr){
		
	} else {
		for clientToHandle.nextClient != nil {
			if (clientToHandle.nextClient.clientAddr == clientAddr){

			}
			clientToHandle = clientToHandle.nextClient
		}
	}

	//client to handle needs to be verified
	var msgFromClient FortuneReqMessage
	err := json.Unmarshal(msg, &msgFromClient)
	if (err != nil){
		return "could not interpret message"
	} else if (msgFromClient.FortuneNonce == clientToHandle.FortuneNonce) {
		fortMsg := FortuneMessage{fortuneString}
		outBuffer, err := json.Marshal(fortMsg)
		checkError(err)
	
		toClient, err := net.ResolveUDPAddr("udp", clientToHandle.clientAddr)
		checkError(err)
	
		_, err = conn.WriteTo(outBuffer, toClient)
		checkError(err)
		return "good"
	} else {
		
		if (checkForWrongAddress(msgFromClient.FortuneNonce, clientAddr)){
			sendErrorBack("unknown remote client address", conn, clientToHandle)
		} else {
			sendErrorBack("incorrect fortune nonce", conn, clientToHandle)
		}
	}
	return "error"
}

//
func checkForWrongAddress(recNonce int64, address string) bool{
	clientToHandle := clientList
	if (clientToHandle.clientAddr != address){
		if (clientToHandle.FortuneNonce == recNonce){
			return true
		}
		
	} else {
		for clientToHandle.nextClient != nil {
			if (clientToHandle.nextClient.clientAddr == address){

			}
			clientToHandle = clientToHandle.nextClient
		}
	}
	return false
}

//go function to handle rpc
func startRPCForClient() {
	fRPC := new(FortuneServerRPC)
	rpc.Register(fRPC)

	tcpResolved, _ := net.ResolveTCPAddr("tcp", fserverRPCAddr)
	incomingConn, err := net.ListenTCP("tcp", tcpResolved)
	checkError(err)
	for{
		acceptedConn, _ := incomingConn.AcceptTCP()
	
		rpc.ServeConn(acceptedConn)
		acceptedConn.Close()
	}
	
	incomingConn.Close()
}

// select which client we are interacting with
func checkHandledClient(clientAddr string) *Client {
	clientToHandle := clientList	
	
	if (clientList == nil){
		
		
		clientList = &Client{clientAddr,-1,nil}
		
		clientToHandle = clientList
		return clientToHandle
	} else if (clientToHandle.clientAddr == clientAddr){
		return clientToHandle
	} else {
		for clientToHandle.nextClient != nil {
			if (clientToHandle.nextClient.clientAddr == clientAddr){
				return clientToHandle.nextClient
			}
			clientToHandle = clientToHandle.nextClient
		}
		
		clientToHandle.nextClient = &Client{clientAddr,-1,nil}
		clientToHandle = clientToHandle.nextClient
	}
		
	return clientToHandle
}

//generate and send aserver nonce
func generateNonce(client **Client){
	var nonce int64
	nonce = rand.Int63n(9223372036854000000)
	(*client).FortuneNonce = nonce
}

//RPC
func (this *FortuneServerRPC) GetFortuneInfo(clientAddr string, fInfoMsg *FortuneInfoMessage) error{
	
	client := checkHandledClient(clientAddr)
	generateNonce(&client)

	fInfoMsg.FortuneServer = fserverUDPString
	fInfoMsg.FortuneNonce = client.FortuneNonce
	return nil
}

// If err is non-nil, print it out and halt.
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}