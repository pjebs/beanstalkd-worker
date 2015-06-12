package main

//Information about sending SMS: https://docs.nexmo.com/index.php/sms-api/send-message
const (
	NEXMO_KEY               = "NEXMO_KEY_HERE"
	NEXMO_SECRET            = "NEXMO_SECRET_HERE"
	NEXMO_REST_API_BASE_URL = "https://rest.nexmo.com/sms/json"
)

// When running app on mac via a linux virtual machine,
// you must change beanstalkd address to 0.0.0.0
// sudo nano /etc/default/beanstalkd
// sudo service beanstalkd restart

const (
	BEANSTALKD_ADDRESS_AND_PORT = "localhost:11300"
	NUM_WORKERS_MULTIPLIER      = 4
)
