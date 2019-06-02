package main

import (
	"os"
	"fmt"
	"math/rand"
	"time"
	"net"
	"bufio"
	"strconv"
	"errors"
	"bytes"
	"strings"

	"github.com/nickeljew/file-relay/filerelay"
)


var (
	ErrNoServer = errors.New("No server")

	// ErrCacheMiss means that a Get failed because the data-key wasn't present.
	ErrCacheMiss = errors.New("Cache miss")

	// ErrCASConflict means that a CompareAndSwap call failed due to the
	// cached value being modified between the Get and the CompareAndSwap.
	// If the cached value was simply evicted rather than replaced,
	// ErrNotStored will be returned instead.
	ErrCASConflict = errors.New("Compare-and-swap conflict")

	// ErrNotStored means that a conditional write operation (i.e. Add or
	// CompareAndSwap) failed because the condition was not satisfied.
	ErrNotStored = errors.New("Data not stored")

	// ErrServer means that a server error occurred.
	ErrServerError = errors.New("Server error")

	// ErrMalformedKey is returned when an invalid key is used.
	// Keys must be at maximum 250 bytes long and not
	// contain whitespace or control characters.
	ErrMalformedKey = errors.New("malformed: key is too long or contains invalid characters")
)


//
func main() {
	fmt.Println("File-Relay client")

	rand.Seed(time.Now().Unix())

	cnt := 1
	fin := make(chan int)

	for i := 0; i < cnt; i++ {
		go doTry(i, fin)
	}
	
	for {
		select {
		case <- fin:
			cnt--
			fmt.Println("- left count: ", cnt)
			if cnt == 0 {
				os.Exit(0)
			}
		}
	}
}


//
type Client struct {
	nc net.Conn
	rw *bufio.ReadWriter
}

func doTry(idx int, fin chan int) {
	if err := trySet(); err != nil {
		fmt.Printf("Error in %d: %s\n", idx, err.Error())
	} else {
		fmt.Printf("Finish %d\n", idx)
	}
	fin <- idx
}

//
func trySet() error {
	cfg, _ := filerelay.InitClientConfig("localhost")
	nc, err := net.Dial("tcp", cfg.Addr())
	if err != nil {
		fmt.Println("Failed connection to: ", cfg.Addr())
		return ErrNoServer
	}

	// reader := bufio.NewReader(os.Stdin)
    // fmt.Print("Text to send: ")
	// text, _ := reader.ReadString('\n')

	client := &Client{
		nc: nc,
		rw: bufio.NewReadWriter(bufio.NewReader(nc), bufio.NewWriter(nc)),
	}

	reqValue := []byte("Hello World - " + strconv.Itoa(random(10, 50)) + "\n" + textsForTest())
	reqline := &filerelay.ReqLine{
		Cmd: "set",
		Key: "test123",
		ValueLen: len(reqValue),
		Flags: 1,
		Expiration: 1800,
	}

	// if !ValidKey(reqline.Key) {
	// 	return ErrMalformedKey
	// }

	toSend := fmt.Sprintf("%s %s %d %d %d\r\n",
	reqline.Cmd, reqline.Key, reqline.Flags, reqline.Expiration, reqline.ValueLen)
	fmt.Printf("Sending:\n>%s", toSend)
	if _, err = fmt.Fprintf(client.rw, toSend); err != nil {
		return err
	}

	if _, err = client.rw.Write(reqValue); err != nil {
		return err
	}
	if _, err := client.rw.Write(filerelay.Crlf); err != nil {
		return err
	}
	if err := client.rw.Flush(); err != nil {
		return err
	}

	line, err := client.rw.ReadSlice('\n')
	if err != nil {
		return err
	}
	fmt.Println("Response from server:", strings.Trim(string(line), " \r\n"))

	switch {
	case bytes.Equal(line, filerelay.ResultStored):
		return nil
	case bytes.Equal(line, filerelay.ResultNotStored):
		return ErrNotStored
	case bytes.Equal(line, filerelay.ResultExists):
		return ErrCASConflict
	case bytes.Equal(line, filerelay.ResultNotFound):
		return ErrCacheMiss
	}

	return fmt.Errorf("Unexpected response line from %q: %q", reqline.Cmd, string(line))
}


func random(min, max int) int {
	return rand.Intn(max-min) + min
}


func textsForTest() string {
	return `
Lewis Hamilton won the Monaco Grand Prix with a perfectly measured drive - but was it really the "miracle" he talked about on the radio?

Hamilton was clearly struggling with his medium-compound tyres, particularly his left front, as he held on against Max Verstappen's Red Bull in a race that had us all on the edge of our seats until the very end.

Inside the car, it must have been hugely uncomfortable for Hamilton. Of all the drivers he could have been holding on against, Verstappen was probably the one he wanted behind him the least.

The Dutchman is always so aggressive, and the fact he had a five-second penalty for an unsafe release put an extra element into the mix. Verstappen knew he had to pass Hamilton or he wouldn't even finish on the podium.

That meant the risk-versus-reward balance of passing in Monaco was edging more towards risk for him.

Hamilton holds off Verstappen in Monaco
The hardest race I've had - Hamilton
"We need a miracle" - Hamilton frustrated with tyre strategy
The role of Hamilton's radio
Lewis Hamilton and Max Verstappen
Lewis Hamilton described his victory in Monaco as a "miracle" on the team radio
Hearing Hamilton's team radio undoubtedly made the race more exciting. He was frequently in discussion with his engineers, saying how difficult his job was and how he would never be able to do it.

No matter what they came back with, Hamilton was insistent for a long time that this was a race he couldn't win.

There are two ways to look at this.

On the one hand, Hamilton was clearly uncomfortable in his car, and was not happy having Verstappen pressuring him so closely for the majority of the race.

Any slip from Hamilton and Verstappen would have been through. Had he got into the lead on track, Verstappen's pace advantage meant he would surely have made up enough time out front to win the race, even with his penalty.

With victory in the most prestigious race of the season at stake, it's perhaps unsurprising he sounded so anxious on the radio.

For me, though, there was an element of showmanship in Hamilton's radio messages. It was all a bit 'Hollywood'.

No other driver would have been on the radio in quite the dramatic way Hamilton was. In fact, we barely heard any other team radio throughout the race, apart from the odd gee-up from Verstappen's race engineer, in typically casual fashion.

Hamilton - That was for Niki
Five things to know about F1's Niki Lauda
Tributes paid to F1 legend Lauda
Making a drama out of a crisis
You could of course understand Hamilton questioning the tyre choice his Mercedes team made.

Pitting Hamilton on to medium tyres instead of the hard, which both Ferrari and Red Bull opted for, made his life more difficult.

If Hamilton had been on hard tyres, he would have easily had the tyre life to get to the end, and we almost certainly wouldn't have witnessed such a race.

But having made his point and questioned it, the constant drama was escalating across the radio as the laps ticked by.

Max Verstappen
Max Verstappen dropped from second on the road to fourth in the results because of a five-second penalty
Because nothing appeared to be changing - Hamilton still had Verstappen at a close, yet comfortable distance - the world champion's showmanship out front really added something to the grand prix.

We were all thinking: 'Surely he can't do it; he knows he can't do it.' It had the ingredients of a Monaco classic.

But in reality, Hamilton still had the race under control.

Without a massive car advantage and a huge amount of risk, you can't pass at Monaco.

Ferrari driver Sebastian Vettel in third and Hamilton's team-mate Valtteri Bottas in fourth knew that for the entire race. They didn't seem to even think about giving it a go as they just circulated within five seconds of Verstappen for the 62 laps post-safety car and waited to head to the podium at the end.

And while I really admire Verstappen for giving it a go, he never really got close enough to seriously challenge.

Monaco farce is no laughing matter for Ferrari
How the Monaco Grand Prix unfolded
How Hamilton stayed in control
The problem Hamilton had was with his front tyres, not his rears. That meant he could always get perfect traction off the corners and disappear on the straights in the crucial places - into the chicane and into Sainte Devote.

Verstappen tried his best - his lunge at the Nouvelle Chicane towards the end was a commendable attempt at doing the impossible - but it was never going to work.

It was clear that Verstappen was driving on the limit in his attempts to engineer an opportunity. He had a few lock-ups, he was sliding on traction, he even had to cut the second part of the Swimming Pool chicane once because he had a huge slide on the entry. But Hamilton never even so much as locked a wheel under this mountain of pressure.

On the one hand it goes to show how well he was dealing with it, but on the other it shows how in relative control he was, and his pace throughout the race backs that notion up.

Just how slowly was Hamilton going?
As soon as the safety car came in and the race restarted, Hamilton was lapping unbelievably slowly. At first he was two seconds per lap off Racing Point's Sergio Perez back in 17th place. Then even George Russell's Williams was a second faster than him, despite being on a harder tyre and qualifying over three seconds slower than his compatriot in the slowest car on the grid.

Hamilton knew what he was doing. He was controlling the pace because the only way he could lose the race was if he pushed too hard and damaged his tyres.

I don't think Hamilton ever really truly pushed during the grand prix, which is why he was so consistent with his driving, and probably why he was able to narrate the story of the race so well with his team over the radio.
`
}


