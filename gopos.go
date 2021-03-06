package main

import (
	"code.google.com/p/gorilla/pat"
	"encoding/hex"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

var debug = flag.Bool("d", false, "set the debug modus( print informations )")
var remserialPort = flag.String("rsport", "23000", "remserial Port [default=23000]")
var remserialIP = flag.String("rsip", "127.0.0.1", "remserial IP [default=127.0.0.1]")

var cn io.Writer
var err error

type Page struct {
	Title string
	Body  []byte
}
type escpos struct {
	dst   io.Writer
	under int
	bold  int
	bw    int
}

var myPrinter *escpos

func (e escpos) SetDst(dst1 io.Writer) {
	e.dst = dst1
}

func (e *escpos) Write(data []byte) (n int, err error) {
	log.Printf("Drucke %s Hex %s\n", data, hex.Dump(data))
	fmt.Fprint(cn, string(data))
	return 0, nil
}

func (e *escpos) toggleBW() {
	if e.bw == 1 {
		e.bw = 0
	} else {
		e.bw = 1
	}
	t := fmt.Sprintf("\x1DB%c", e.bw)
	e.Send(t)
}

func (e escpos) init() {
	e.Send("\x1B\x40")
}

func (e escpos) cut() {
	e.Send("\x1DVA0")
}

func (e escpos) ff() {
	e.Send("\n")
}
func (e escpos) ffn(n int) {
	t := fmt.Sprintf("\x1Bd%c", n)
	e.Send(t)
}
func (e escpos) codePage858() {
	t := fmt.Sprintf("\x1BR%c", 2)
	e.Send(t)
}
func (e escpos) fontA() {
	e.Send("\x1BM0")
}
func (e escpos) fontB() {
	e.Send("\x1BM1")
}
func (e escpos) fontC() {
	e.Send("\x1BM2")
}

func (e escpos) doubleStrike() {
	e.Send("\x1BG\x01")
}
func (e *escpos) toggleBold() {
	if e.bold == 1 {
		e.bold = 0
	} else {
		e.bold = 1
	}
	t := fmt.Sprintf("\x1BG%c", e.bold)
	e.Send(t)
}

func (e escpos) underline() {
	e.Send("\x1B-\x01")
	e.under = 1
}

func (e *escpos) toggleUnderline() {
	if e.under == 1 {
		e.under = 0
	} else {
		e.under = 1
	}
	t := fmt.Sprintf("\x1B-%c", e.under)
	e.Send(t)
}

func (e escpos) left() {
	e.Send("\x1Ba\x00")
}

func (e escpos) centre() {
	e.Send("\x1Ba\x01")
}

func (e escpos) right() {
	e.Send("\x1Ba\x02")
}

func (e escpos) reallywide() {
	e.Send("\x1D\x21\x70")
}

func (e escpos) normalwide() {
	e.Send("\x1D\x21\x00")
}

// func Log(v ...): loging. give log information if debug is true
func Log(v ...interface{}) {
	if *debug == true {
		ret := fmt.Sprint(v)
		log.Printf("escpos: %s\n", ret)
	}
}

// func test(): testing for error
func Test(err error, mesg string) {
	if err != nil {
		log.Print("CLIENT: ERROR: ", mesg)
		os.Exit(-1)
	} else {
		Log("Ok: ", mesg)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	title := "Print"
	//	log.Print("handler(): Method is ", r.Method)
	p := &Page{Title: title}
	t, _ := template.ParseFiles("print.html")
	t.Execute(w, p)
	log.Print("handler(): serving print.html")
	return

}

func printHandler(w http.ResponseWriter, r *http.Request) {

	// init to reset all settings
	myPrinter.init()

	body := r.FormValue("body")
	log.Print("handler(): body is ", body)

	// look for special formatter
	// * bold
	// _ underline
	// { left justified
	// } right justified
	// ^ center
	// - invers
	b := strings.FieldsFunc(body, func(char rune) bool {
		switch char {
		case '*', '-', '_', '{', '}', '^':
			return true
		}
		return false
	})
	lange := 0
	for _, data := range b {
		// myPrinter.Send(data)
		fmt.Fprint(myPrinter, data)
		fmt.Printf(" Drucke <%s> \n", data)
		lange += len(data)
		if lange < len(body) {
			fmt.Printf(" Drucke <%s> Trenner %c lange %i\n", data, body[lange], lange)

			switch body[lange] {
			case '*':
				myPrinter.toggleBold()
			case '_':
				myPrinter.toggleUnderline()
			case '-':
				myPrinter.toggleBW()
			case '{':
				myPrinter.left()
			case '}':
				myPrinter.right()
			case '^':
				myPrinter.centre()
			}
			lange += 1
		}
	}

	// beautify: print 5 form feeds 
	myPrinter.ffn(5)

	//cut paper
	myPrinter.cut()

	http.Redirect(w, r, "/", http.StatusFound)
}

var Usage = func() {
	fmt.Fprintf(os.Stderr, "\nUsage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n\n")
}

func main() {

	flag.Usage = Usage
	flag.Parse()

	// connect to remserial
	destination := *remserialIP + ":" + *remserialPort // "127.0.0.1:23000"
	log.Print("main(): connect to remserial via ", destination)
	cn, err = net.Dial("tcp", destination)
	Test(err, "connecting....")

	// initialize printer object
	myPrinter = new(escpos)

	//set io.Writer Interface to printer
	myPrinter.SetDst(cn)

	// Setup router
	r := pat.New()

	// Handler for POST Requests is printHandler
	r.Post("/print", printHandler)

	// Normal handler
	r.Get("/", handler)

	// Routing all traffic to router
	http.Handle("/", r)

	// "main loop"
	http.ListenAndServe(":8080", nil)

}
