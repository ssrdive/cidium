package models

import (
	"fmt"
	"math"
	"time"
)

type Installment struct {
	Capital         float64 `json:"capital"`
	Interest        float64 `json:"interest"`
	DefaultInterest float64 `json:"default_interest"`
	DueDate         string  `json:"due_date"`
}

func Create(capital, rate float64, installments, installmentInterval int, initiationDate, method string) ([]Installment, error) {
	initDate, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s 00:00:00", initiationDate))
	if err != nil {
		return nil, err
	}

	installmentCapital := math.Round((capital/float64(installments))*100) / 100
	schedule := make([]Installment, installments)

	if method == "S" {
		realRate := rate * 0.01
		interest := (realRate / float64(12)) * float64(installmentInterval) * float64(installments) * capital
		instInterest := math.Round((interest/float64(installments))*100) / 100
		for i := 0; i < installments; i++ {
			initDate = initDate.AddDate(0, installmentInterval, 0)
			schedule[i] = Installment{
				Capital:         installmentCapital,
				Interest:        instInterest,
				DefaultInterest: 0,
				DueDate:         initDate.Format("2006-01-02 15:04:05"),
			}
		}
	} else if method == "R" {
		instInterest := (rate / (float64(12) / float64(installmentInterval))) * 0.01
		for i := 0; i < installments; i++ {
			initDate = initDate.AddDate(0, installmentInterval, 0)
			schedule[i] = Installment{
				Capital:         installmentCapital,
				Interest:        math.Round(((capital-(installmentCapital*float64(i)))*instInterest)*100) / 100,
				DefaultInterest: 0,
				DueDate:         initDate.Format("2006-01-02 15:04:05"),
			}
		}
	} else if method == "IRR" {
		P := capital
		r := rate / float64(12/installmentInterval) / 100
		n := installments

		payment := math.Round((P*r*(math.Pow(1+r, float64(n))/(math.Pow(1+r, float64(n))-1)))*100) / 100

		for i := 1; i <= n; i++ {
			initDate = initDate.AddDate(0, installmentInterval, 0)
			rentalInterest := math.Round((((P*r)-payment)*math.Pow((r+1), (float64(i)-1))+payment)*100) / 100
			rentalCapital := math.Round((payment-rentalInterest)*100) / 100
			schedule[i-1] = Installment{
				Capital:         rentalCapital,
				Interest:        rentalInterest,
				DefaultInterest: 0,
				DueDate:         initDate.Format("2006-01-02 15:04:05"),
			}
		}
	}

	capitalTotal := installmentCapital * float64(installments)
	capitalDiff := math.Round((capital-capitalTotal)*100) / 100
	schedule[installments-1].Capital = math.Round((schedule[installments-1].Capital+capitalDiff)*100) / 100

	return schedule, nil
}
