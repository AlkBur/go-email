package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/lib/pq"
	"golang.org/x/text/encoding/charmap"

	imap "github.com/emersion/go-imap"
	clientIMAP "github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type UserJSON struct {
	Email string `json:"email"`
	Password string `json:"password"`
	IsAdmin bool `json:"admin"`
}

type Product struct {
	Name    string `json:"name"`
	ID string `json:"id" gorm:"primary_key"`
	Price pq.Float64Array `json:"price" gorm:"type:double[]"`
}

func main() {
	var login string    
    var password string  
	var fileIn string  
	var fileOut string 
	var date time.Time 
 
    // flags declaration using flag package
    flag.StringVar(&login, "u", "", "Specify username. Default is admin")
    flag.StringVar(&password, "p", "", "Specify pass. Default is admin")
	flag.StringVar(&fileIn, "in", "", "Specify in file. Default is in.txt")
	flag.StringVar(&fileOut, "out", "", "Specify out file. Default is out.txt")
	flag.Func("d", "data emails", func(flagValue string) error {
		if t, err := time.Parse("2006-01-02", flagValue); err != nil {
			date = time.Now()
		} else {
			date = t
		}
		return nil
    })
    flag.Parse()  // after declaring flags we need to call it

	if fileIn != "" {
		PostProdacts(login, password, fileIn)
	}
	if fileOut != "" {
		GetEmail(login, password, fileOut, date)
	}
}

func GetEmail(login, password, filename string, date time.Time) {
	y1, m1, d1 := date.Date()
	// Connect to server
	c, err := clientIMAP.DialTLS("imap.mail.ru:993", &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	// Don't forget to logout
	defer c.Logout()

	// Login
	if err := c.Login(login, password); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")

	// List mailboxes
	// mailboxes := make(chan *imap.MailboxInfo, 10)
	// done := make(chan error, 1)
	// go func () {
	// 	done <- c.List("", "*", mailboxes)
	// }()

	// log.Println("Mailboxes:")
	// for m := range mailboxes {
	// 	log.Println("* " + m.Name)
	// }

	// if err := <-done; err != nil {
	// 	log.Fatal(err)
	// }

	// Select INBOX
	mbox, err := c.Select("INBOX/ToMyself", false)
	if err != nil {
		log.Fatal(err)
	}
	//log.Println("Flags for INBOX:", mbox.Flags)

	// Get the last 4 messages
	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > 200 {
		// We're using unsigned integers here, only subtract if the result is > 0
		from = mbox.Messages - 200
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	var section imap.BodySectionName

	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{section.FetchItem()}, messages)
	}()

	log.Println("messages:")
	for msg := range messages {
		
		//log.Println("* " + msg.Envelope.Subject)
		//log.Println("* " + fmt.Sprintf("%v", msg.Envelope.Date))	
		
		r := msg.GetBody(&section)
		if r == nil {
			log.Println("Server didn't returned message body")
			continue
		}
		// Create a new mail reader
		mr, err := mail.CreateReader(r)
		if err != nil {
			log.Println(err)
			continue
		}

		// Print some info about the message
		header := mr.Header
		if date, err := header.Date(); err == nil {
			y, m, d := date.Date()
			if y!=y1 || m!=m1 || d != d1 {
				//log.Println("error Date:", date)
				continue
			}	
			//log.Println("Date:", date)
		}else{
			continue
		}

		// Process each message's part
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}

			switch p.Header.(type) {
			case *mail.InlineHeader:
				// This is the message's text (can be plain-text or HTML)
				//b, err := ioutil.ReadAll(p.Body)
				// if err != nil {
				// 	log.Fatal(err)
				// 	continue
				// }
				//log.Println(string(b))
				doc, err := goquery.NewDocumentFromReader(p.Body)
				//emailTag, err := htmlparse.NewParser(b).Parse()
				if err != nil {
				 	log.Println(err)
				 	continue
				}
				//log.Println("h2")
				h2 := doc.Find("h2")
				h2 = h2.Parent()
				h2 = h2.Parent()
				h2 = h2.Parent()
				h2 = h2.Find("p")
				text := strings.TrimSpace(h2.Text())

				res := strings.LastIndex(text, " ")
				if res > 0 {
					text = text[res+1:]
					log.Printf("%v\n", text)


					h2 = doc.Find("#order p")
					f := func(i int, s *goquery.Selection) {
						text = strings.TrimSpace(s.Text())
						log.Printf("%v\n", text)
					}
					h2.Each(f)

				}

				// doc.Find("h2").Each(func(i int, s *goquery.Selection) {
				// 	// For each item found, get the title
				// 	title := s.Text()
				// 	log.Printf("Review %d: %s\n", i, title)
				// })
				//originalPrices := emailTag.FindByClass("h2")
				//log.Println(originalPrices)

				// doc, err := html.Parse(p.Body)
				// if err != nil {
				// 	log.Println(err)
				// 	continue
				// }
				// fmt.Println(doc)
				// z := html.NewTokenizer(p.Body)
				// for {
				// 	tt := z.Next()
				
				// 	switch {
				// 	case tt == html.ErrorToken:
				// 		// End of the document, we're done
				// 		return
				// 	case tt == html.StartTagToken:
				// 		t := z.Token()
				
				// 		//isAnchor := t.Data == "tr"
				// 		//if isAnchor {
				// 			fmt.Println(t.Data)
				// 		//}
				// 	}
				// }

				//log.Println("Got text: %v", string(b))
			// case *mail.AttachmentHeader:
			// 	// This is an attachment
			// 	filename, _ := h.Filename()
			// 	log.Println("Got attachment: %v", filename)
			}
		}
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	log.Println("Done!")


	return




	// Запись строки в кодировке Windows-1252
	encoder := charmap.Windows1252.NewEncoder()
	s, e := encoder.String("This is sample text with runes Š")
	if e != nil {
		panic(e)
	}
	ioutil.WriteFile(filename, []byte(s), os.ModePerm)
}

func PostProdacts(login, password, filename string) {
	user := &UserJSON{
		Email:    login,
		Password: password,
		IsAdmin: true,
	}

	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(user)


	server := `https://dsp-shop.tk`
	link := server + `/login`

    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    client := &http.Client{Transport: tr}


	req, _ := http.NewRequest("POST", link, buf)
    res, err := client.Do(req)
    if err != nil {
        log.Fatalln(err)
    }

	defer res.Body.Close()

	data := make(map[string]interface{})
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
        log.Fatalln(err)
    }

	// Декодировка в UTF-8
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	//Пропустим boom
	f.Seek(3, 0)

	// decoder := charmap.Windows1251.NewDecoder()
	// reader := decoder.Reader(f)
	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalln(err)
	}
	//fmt.Println(string(b))



	goods := make([]Product, 0)
	//plan, _ := ioutil.ReadFile(filename)
	err = json.Unmarshal(b, &goods)
	if err != nil {
        log.Fatalln(err)
    }

	//log.Println(goods)


	link = server + `/api/prodacts`
	json.NewEncoder(buf).Encode(goods)

	var bearer = "Bearer " + data["token"].(string)
	req, _ = http.NewRequest("POST", link, buf)
	req.Header.Add("Authorization", bearer)
	res, err = client.Do(req)
    if err != nil {
        log.Fatalln("Error on response.\n[ERROR] -", err)
    }
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
    if err != nil {
        log.Println("Error while reading the response bytes:", err)
    }
    log.Println(string([]byte(body)))

}