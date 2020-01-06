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
	} else {
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
	}

	capitalTotal := installmentCapital * float64(installments)
	capitalDiff := math.Round((capital-capitalTotal)*100) / 100
	schedule[installments-1].Capital = math.Round((schedule[installments-1].Capital+capitalDiff)*100) / 100

	return schedule, nil
}
