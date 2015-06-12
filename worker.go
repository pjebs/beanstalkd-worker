package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type sendSMSWorker struct {
}

func (worker *sendSMSWorker) TunnyReady() bool {
	return true
}

// This is where the work actually happens
func (worker *sendSMSWorker) TunnyJob(data interface{}) (r interface{}) {
	/* TODO: Use and modify state
	 * there's no need for thread safety paradigms here unless the
	 * data is being accessed from another goroutine outside of
	 * the pool.
	 */

	input := data.(TunnyInput)
	log.Printf("New Job: %d\n", input.Id)

	defer func(id uint64) {
		err := recover()
		if err != nil {
			r = BeanstalkdIgnore
		}
		log.Printf("Job done: %d\n", input.Id)
	}(input.Id)

	type InputData struct {
		From      string `json:"from"`
		To        string `json:"to"`
		Message   string `json:"message"`
		ClientRef string `json:"client-ref,omitempty"`
		Ttl       string `json:"ttl,omitempty"`
	}

	inputData := &InputData{}
	err := json.Unmarshal(input.Body, inputData)
	if err != nil {
		return BeanstalkdBury
	}

	baseUrl := "https://rest.nexmo.com/sms/json"
	parameters := url.Values{
		"api_key":    {NEXMO_KEY},
		"api_secret": {NEXMO_SECRET},
		"from":       {inputData.From},
		"to":         {inputData.To},
		"text":       {inputData.Message},
	}

	if inputData.ClientRef != "" {
		parameters.Set("client-ref", inputData.ClientRef)
	}

	if inputData.Ttl != "" {
		parameters.Set("ttl", inputData.Ttl)
	}

	resp, err := http.PostForm(baseUrl, parameters)
	if err != nil {
		return BeanstalkdRelease
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return BeanstalkdRelease
	}

	type Message struct {
		Status           string `json:"status"`
		MessageId        string `json:"message-id,omitempty"`
		To               string `json:"to,omitempty"`
		ClientRef        string `json:"client-ref,omitempty"`
		RemainingBalance string `json:"remaining-balance,omitempty"`
		MessagePrice     string `json:"message-price,omitempty"`
		Network          string `json:"network,omitempty"`
		ErrorText        string `json:"error-text,omitempty"`
	}

	type JsonResponse struct {
		MessageCount string    `json:"message-count"`
		Messages     []Message `json:"messages"`
	}

	var jsonResponse JsonResponse
	err = json.Unmarshal(contents, &jsonResponse)
	if err != nil {
		return BeanstalkdRelease
	}

	if jsonResponse.Messages[0].Status == "6" && jsonResponse.Messages[0].ErrorText == "Unroutable message - rejected" {
		log.Printf("Bury Job id: %d response: %+v err: %v\n", input.Id, jsonResponse, err)
		return BeanstalkdBury
	}

	if jsonResponse.Messages[0].Status != "0" {
		//An error occurred - Print to log
		log.Printf("Could not complete Job id: %d response: %+v err: %v\n", input.Id, jsonResponse, err)
		return BeanstalkdRelease
	}

	// log.Printf("Completed Job id: %d response: %+v", input.Id, jsonResponse)

	return BeanstalkdDelete

}
