/*
Usage go run auth-server.go [aserver UDP ip:port] [fserver RPC ip:port] [secret]
*/

package main

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"math/rand"
	"time"
	"net/rpc"
)

/////////// Error messages
type ErrMessage struct {
	Error string
}


/////////// Messages to/from client
//~~~ Message containing a nonce from auth-server.
type NonceMessage struct {
	Nonce int64
}

//~~~ Message containing an MD5 hash from client to auth-server.
type HashMessage struct {
	Hash string
}

//~~~ Message with details for contacting the fortune-server.
type FortuneInfoMessage struct {
	FortuneServer string
	FortuneNonce  int64
}

////////// list of clients and their info
type Client struct{
	clientAddr string
	clientNonce int64
	clientHash string
	FortuneNonce int64
	nextClient *Client
}

var aserverAddr *net.UDPAddr
var secret int64
var fserverAddr string

///Main Method
func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	var clientList *Client = nil

	//process args
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr,
			"Usage: %s [aserver UDP ip:port] [fserver RPC ip:port] [secret]\n",
			os.Args[0])
		os.Exit(1)
	}

	local := os.Args[1]
	aserverAddr, err := net.ResolveUDPAddr("udp", local)
	checkError(err)

	fserverAddr = os.Args[2]
	

	secret, err = strconv.ParseInt(os.Args[3], 10, 64)
	checkError(err)

	//listen for connection
	conn, _ := net.ListenUDP("udp", aserverAddr)
	
	
	for{
		
		
		
		buf := make([]byte, 1024)
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
        		checkError(err)
        		os.Exit(1)
    		}
		
		
		clientToService := checkHandledClient(remoteAddr.String(), &clientList)
		
		go sendMessageToClient(&clientToService, buf[:n], conn)
		
		
		
	}


}

//handle sending messages to client
func sendMessageToClient(client **Client, buf []byte, conn *net.UDPConn){
	var hashResponse HashMessage 
	
		

	json.Unmarshal(buf, &hashResponse)
	if (hashResponse.Hash != "") {
		if(hashResponse.Hash != (*client).clientHash){
			sendError("unexpected hash value", conn, client)
		} else {
			replyMsg := getFortuneNonce((*client).clientAddr)
			sendFortuneServInfo(client, conn, replyMsg)
		}
	} else {
		generateAndSendNonce(client, conn)
	}
	
	
}
//retrieve the fortune server nonce from fserver
func getFortuneNonce(clientAddr string) FortuneInfoMessage {
	
	var replyMessage FortuneInfoMessage
	
	fserverConn, err := rpc.Dial("tcp", fserverAddr)
		checkError(err)
	fserverConn.Call("FortuneServerRPC.GetFortuneInfo", clientAddr, &replyMessage)
	fserverConn.Close()
	return replyMessage
}

//send fortune info to client
func sendFortuneServInfo(client **Client, conn *net.UDPConn, replyMsg FortuneInfoMessage){

	
	outBuffer, err := json.Marshal(replyMsg)
	checkError(err)
	
	toClient, err := net.ResolveUDPAddr("udp", (*client).clientAddr)
	checkError(err)
	
	_, err = conn.WriteTo(outBuffer, toClient)
	checkError(err)
}

//generate and send aserver nonce
func generateAndSendNonce(client **Client, conn *net.UDPConn){
	var nonce int64
	nonce = rand.Int63n(9223372036854000000)
	(*client).clientNonce = nonce
	(*client).clientHash = computeNonceSecretHash(nonce)
	(*client).FortuneNonce = -1

	nonceMsg := NonceMessage{nonce}
	outBuffer, err := json.Marshal(nonceMsg)
	checkError(err)
	
	toClient, err := net.ResolveUDPAddr("udp", (*client).clientAddr)
	checkError(err)
	
	_, err = conn.WriteTo(outBuffer, toClient)
	checkError(err)
	
}


// select which client we are interacting with
func checkHandledClient(clientAddr string, clientList **Client) *Client {
	clientToHandle := *clientList	
	
	if (*clientList == nil){
		
		*clientList = &Client{clientAddr,-1,"",-1,nil}
		
		clientToHandle = *clientList
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
		
		clientToHandle.nextClient = &Client{clientAddr,-1,"",-1,nil}
		clientToHandle = clientToHandle.nextClient
	}
		
	return clientToHandle
}


// Returns the MD5 hash as a hex string for the (nonce + secret) value.
func computeNonceSecretHash(nonce int64) string {
	sum := nonce + secret
	buf := make([]byte, 512)
	n := binary.PutVarint(buf, sum)
	h := md5.New()
	h.Write(buf[:n])
	str := hex.EncodeToString(h.Sum(nil))
	return str
}

// Send error message
func sendError(message string, conn *net.UDPConn, client **Client){
	errorMsg := ErrMessage{message}
	outBuffer, err := json.Marshal(errorMsg)
	checkError(err)
	
	toClient, err := net.ResolveUDPAddr("udp", (*client).clientAddr)
	checkError(err)
	
	_, err = conn.WriteTo(outBuffer, toClient)
	checkError(err)
}	

// If err is non-nil, print it out and halt.
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}