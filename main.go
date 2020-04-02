package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/customer"
)

// PostPaymentResponse is represents what the postPayment route will return
type PostPaymentResponse struct {
	ID     string `json:"id"`
	Amount int64  `json:"amount"`
	Status string `json:"status"`
}

var accountToCustomerLookup map[*string]string = make(map[*string]string)

func postPayment(w http.ResponseWriter, req *http.Request) {
	accountID := stripe.String(req.FormValue("account_id"))
	fmt.Println(req.FormValue("account_id"))
	fmt.Println(req.FormValue("amount"))
	amount, err := strconv.ParseInt(req.FormValue("amount"), 10, 64)
	if err != nil {
		log.Fatal(err)
		panic(fmt.Sprintf("Amount, %v, is invalid", req.FormValue("amount")))
	}

	stripe.Key = "sk_test_4eC39HqLyjWDarjtT1zdp7dc"

	customerID, exists := accountToCustomerLookup[accountID]

	if !exists {
		customerParams := &stripe.CustomerParams{}
		customerParams.SetSource("tok_amex")
		newCustomer, err := customer.New(customerParams)

		if err != nil {
			panic(fmt.Sprintf("There was a problem creating a customer for account, %v. Please making the purchase again.", accountID))
		}

		accountToCustomerLookup[accountID] = newCustomer.ID
	}

	customerID = accountToCustomerLookup[accountID]

	chargeParams := &stripe.ChargeParams{
		Amount:      &amount,
		Currency:    stripe.String(string(stripe.CurrencyUSD)),
		Description: stripe.String("My First Test Charge (created for API docs)"),
		Customer:    &customerID,
	}

	newCharge, _ := charge.New(chargeParams)

	w.Header().Set("Content-Type", "application/json")
	response := PostPaymentResponse{
		ID:     newCharge.ID,
		Amount: newCharge.Amount,
		Status: newCharge.Status,
	}

	json.NewEncoder(w).Encode(response)
}

func getPaymentCollection(w http.ResponseWriter, req *http.Request) {
	accountID := stripe.String(req.FormValue("account_id"))

	customerID := accountToCustomerLookup[accountID]

	params := &stripe.ChargeListParams{
		Customer: &customerID,
	}

	chargesList := charge.List(params)

	for chargesList.Next() {
		fmt.Println(chargesList.Charge())
	}

}

func main() {
	http.HandleFunc("/postPayment", postPayment)

	fmt.Println("Listening...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
