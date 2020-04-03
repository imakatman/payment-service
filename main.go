package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/customer"
)

type error struct {
	Message string `json:"message"`
}

func handleError(w http.ResponseWriter, msg string) {
	log.Println(msg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	response := error{
		Message: msg,
	}
	json.NewEncoder(w).Encode(response)
}

type paymentPayload struct {
	Amount    int64  `json:"amount"`
	AccountID string `json:"account_id"`
}

type payment struct {
	ID     string `json:"id"`
	Amount int64  `json:"amount"`
	Status string `json:"status"`
}

type paymentCollection struct {
	Payments []payment `json:"payments"`
}

var accountToCustomerLookup map[string]string = make(map[string]string)

func postPayment(w http.ResponseWriter, req *http.Request) {
	var newPayment paymentPayload
	
	req.Body = http.MaxBytesReader(w, req.Body, 1048576)
	decoder := json.NewDecoder(req.Body)
	decoderErr := decoder.Decode(&newPayment)
	decoder.DisallowUnknownFields()
	if decoderErr != nil {
		handleError(w, "There was a problem decoding the payload")
		return
	}

	accountID := newPayment.AccountID
	amount := newPayment.Amount

	if amount == 0 {
		handleError(w, "There was a problem with the payment amount. Please make sure that the amount is at least $0.50 and or that the payload is correct. The expected type is a string in cents.")
		return
	}

	stripe.Key = "sk_test_4eC39HqLyjWDarjtT1zdp7dc"
	customerID, customerExists := accountToCustomerLookup[accountID]
	if !customerExists {
		customerParams := &stripe.CustomerParams{}
		customerParams.SetSource("tok_amex")
		newCustomer, err := customer.New(customerParams)

		if err != nil {
			errorMessage := fmt.Sprintf("There was a problem creating a customer for account, %v. Please try making the purchase again.", accountID)
			handleError(w, errorMessage)
			return
		}

		accountToCustomerLookup[accountID] = newCustomer.ID
	}

	customerID = accountToCustomerLookup[accountID]

	chargeParams := &stripe.ChargeParams{
		Amount:   &amount,
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		Customer: &customerID,
	}

	newCharge, newChargeErr := charge.New(chargeParams)
	if newChargeErr != nil {
		errorMessage := fmt.Sprintf("There was a problem charging account, %v. Please try charging again.", accountID)
		handleError(w, errorMessage)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := payment{
		ID:     newCharge.ID,
		Amount: newCharge.Amount,
		Status: newCharge.Status,
	}
	json.NewEncoder(w).Encode(response)
}

func getPaymentCollection(w http.ResponseWriter, req *http.Request) {
	accountID := strings.TrimPrefix(strings.TrimSuffix(req.URL.Path, "/payments"), "/")
	customerID, customerExists := accountToCustomerLookup[accountID]

	if !customerExists {
		handleError(w, "This account either doesn't exist or doesn't have any posted payments.")
		return
	}

	var payments []payment
	chargeListParams := &stripe.ChargeListParams{
		Customer: &customerID,
	}
	chargesList := charge.List(chargeListParams)

	for chargesList.Next() {
		payments = append(payments, payment{
			ID:     chargesList.Charge().ID,
			Amount: chargesList.Charge().Amount,
			Status: chargesList.Charge().Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	response := paymentCollection{
		Payments: payments,
	}
	json.NewEncoder(w).Encode(response)
}

// handleRoot can only handle /:account_id/payments right now
func handleRoot(w http.ResponseWriter, req *http.Request) {
	// Check the request path to make sure it is not too deep
	forwardSlashRune := '/'
	pathInBytes := []rune(req.URL.Path)
	forwardSlashByteAcc := 0

	for _, b := range pathInBytes {
		if b == forwardSlashRune {
			forwardSlashByteAcc++
		}
	}

	pathIsNotTooDeep := forwardSlashByteAcc < 3

	if strings.HasSuffix(req.URL.Path, "/payments") && pathIsNotTooDeep {
		getPaymentCollection(w, req)
		return
	}

	handleError(w, "The requested URL does not exist.")
}

func main() {
	http.HandleFunc("/postPayment", postPayment)
	http.HandleFunc("/", handleRoot)

	log.Println("Listening on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
