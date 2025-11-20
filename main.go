package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

const accountBalanceFile = "balance.txt"

func main() {
	// Load initial balance from file
	accountBalance, err := getBalanceFromFile()
	if err != nil {
		fmt.Println(err)
	}

	for {
		fmt.Println("\nWelcome to Our Bank")
		fmt.Println("1. Check Amount")
		fmt.Println("2. Deposit Amount")
		fmt.Println("3. Withdraw Amount")
		fmt.Println("4. Exit")

		fmt.Print("Enter your choice: ")
		var choice int
		fmt.Scan(&choice)

		switch choice {
		case 1:
			fmt.Println("Your Balance:", accountBalance)

		case 2:
			var deposit float32
			fmt.Print("Enter Amount: ")
			fmt.Scan(&deposit)

			if deposit < 0 {
				fmt.Println("❌ You cannot deposit a negative amount.")
				continue
			}

			accountBalance += deposit
			writeBalanceToFile(accountBalance)
			fmt.Println("✅ Your new balance:", accountBalance)

		case 3:
			var withdrawal float32
			fmt.Print("Enter Amount to Withdraw: ")
			fmt.Scan(&withdrawal)

			if withdrawal <= 0 {
				fmt.Println("❌ Invalid withdrawal amount.")
				continue
			}

			if withdrawal > accountBalance {
				fmt.Println("❌ You cannot withdraw more than your balance.")
				continue
			}

			accountBalance -= withdrawal
			writeBalanceToFile(accountBalance)
			fmt.Println("💰 Your balance after withdrawal:", accountBalance)

		case 4:
			fmt.Println("🙏 Thank you for banking with us!")
			return

		default:
			fmt.Println("❌ Invalid choice. Please try again.")
		}
	}
}

func writeBalanceToFile(balance float32) {
	balanceText := fmt.Sprint(balance)
	os.WriteFile(accountBalanceFile, []byte(balanceText), 0644)
}

func getBalanceFromFile() (float32, error) {
	data, err := os.ReadFile(accountBalanceFile)
	if err != nil {
		return 1000, errors.New("No stored balance found, starting with 1000")
	}

	balanceText := string(data)

	balance, err := strconv.ParseFloat(balanceText, 64)
	if err != nil {
		return 1000, errors.New("Failed to parse stored value")
	}

	return float32(balance), nil
}
