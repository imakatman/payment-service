package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/customer"
)

type error struct {
	Message string `json:"message"`
}

func handleError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	response := error{
		Message: "The requested URL does not exist.",
	}
	json.NewEncoder(w).Encode(response)
	w.WriteHeader(http.StatusBadRequest)
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
	accountID := req.FormValue("account_id")
	// @TODO: Amount should be able to come in as floating numbers
	amount, err := strconv.ParseInt(req.FormValue("amount"), 10, 64)
	if err != nil {
		log.Fatal(err)
		panic(fmt.Sprintf("Amount, %v, is invalid", req.FormValue("amount")))
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
	accountID := strings.TrimPrefix(req.URL.Path, "/payments/")
	customerID, customerExists := accountToCustomerLookup[accountID]

	if !customerExists {
		handleError(w, "This account does not exist.")
		return
	}

	var payments []payment
	params := &stripe.ChargeListParams{
		Customer: &customerID,
	}
	chargesList := charge.List(params)

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

func handleRoot(w http.ResponseWriter, req *http.Request) {
	// @TODO: Make sure there are no more paths
	if strings.HasPrefix(req.URL.Path, "/payments") {
		getPaymentCollection(w, req)
		return
	}

	handleError(w, "The requested URL does not exist.")
}

func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/postPayment", postPayment)

	fmt.Println("Listening on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
