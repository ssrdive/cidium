package loan

import (
	"fmt"
	"math"
	"time"
)

// Installment holds an installment of the marketed rental schedule
type Installment struct {
	Capital         float64 `json:"capital"`
	Interest        float64 `json:"interest"`
	DefaultInterest float64 `json:"default_interest"`
	DueDate         string  `json:"due_date"`
}

// InstallmentSchedule holds an installment of the financial rental schedule
type InstallmentSchedule struct {
	Capital             float64 `json:"capital"`
	Interest            float64 `json:"interest"`
	MonthlyDate         string  `json:"due_date"`
	MarketedInstallment int     `json:"marketed_installment"`
	MarketedCapital     float64 `json:"marketed_capital"`
	MarketedInterest    float64 `json:"marketed_interest"`
	MarketedDueDate     string  `json:"marketed_due_date"`
}

// Create creates marketed and financial rental schedule
func Create(capital, rate float64, installments, installmentInterval, structuredMonthlyRental int, initiationDate, method string) ([]Installment, []InstallmentSchedule, error) {
	initDate, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s 00:00:00", initiationDate))
	if initDate.Day() > 28 {
		initDate = initDate.AddDate(0, 0, -(initDate.Day() - 28))
	}
	if err != nil {
		return nil, nil, err
	}

	installmentCapital := math.Round((capital/float64(installments))*100) / 100
	marketedSchedule := make([]Installment, installments)
	financialSchedule := make([]InstallmentSchedule, installmentInterval*installments)

	if method == "R2" && structuredMonthlyRental > 0 && installmentInterval > 2 {
		marketedSchedule = make([]Installment, installments*installmentInterval)

		installmentCapital = installmentCapital - float64((structuredMonthlyRental * (installmentInterval - 1)))

		P := capital
		r := rate / float64(12) / 100
		n := installmentInterval * installments

		payment := math.Round((P*r*(math.Pow(1+r, float64(n))/(math.Pow(1+r, float64(n))-1)))*100) / 100

		capitalTotal := float64(0)
		// instInterest := (rate / (float64(12) / float64(installmentInterval))) * 0.01
		for i := 1; i <= n; i++ {
			if i%installmentInterval == 0 {
				instInterest := float64(0)
				for j := (i - installmentInterval) + 1; j <= i; j++ {
					rentalInterest := math.Round((((P*r)-payment)*math.Pow((r+1), (float64(j)-1))+payment)*100) / 100
					instInterest = instInterest + rentalInterest
				}

				initDate = initDate.AddDate(0, 1, 0)
				marketedSchedule[i-1] = Installment{
					Capital:         installmentCapital,
					Interest:        math.Round(instInterest*100) / 100,
					DefaultInterest: 0,
					DueDate:         initDate.Format("2006-01-02"),
				}
			} else {
				capitalTotal += float64(structuredMonthlyRental)
				initDate = initDate.AddDate(0, 1, 0)
				marketedSchedule[i-1] = Installment{
					Capital:         float64(structuredMonthlyRental),
					Interest:        0,
					DefaultInterest: 0,
					DueDate:         initDate.Format("2006-01-02"),
				}
			}
		}

		capitalTotal += installmentCapital * float64(installments)
		capitalDiff := math.Round((capital-capitalTotal)*100) / 100
		marketedSchedule[n-1].Capital = math.Round((marketedSchedule[n-1].Capital+capitalDiff)*100) / 100

		initDate, err = time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s 00:00:00", initiationDate))
		if initDate.Day() > 28 {
			initDate = initDate.AddDate(0, 0, -(initDate.Day() - 28))
		}
		capitalTotal = float64(0)
		for i := 1; i <= n; i++ {
			initDate = initDate.AddDate(0, 1, 0)
			rentalInterest := math.Round((((P*r)-payment)*math.Pow((r+1), (float64(i)-1))+payment)*100) / 100
			rentalCapital := math.Round((payment-rentalInterest)*100) / 100
			capitalTotal = capitalTotal + rentalCapital
			financialSchedule[i-1] = InstallmentSchedule{
				Capital:     rentalCapital,
				Interest:    rentalInterest,
				MonthlyDate: initDate.Format("2006-01-02"),
			}

			financialSchedule[i-1].MarketedInstallment = 1
			financialSchedule[i-1].MarketedCapital = marketedSchedule[i-1].Capital
			financialSchedule[i-1].MarketedInterest = marketedSchedule[i-1].Interest
			financialSchedule[i-1].MarketedDueDate = marketedSchedule[i-1].DueDate
		}
		capitalDiff = math.Round((P-capitalTotal)*100) / 100
		financialSchedule[n-1].Capital = math.Round((financialSchedule[n-1].Capital+capitalDiff)*100) / 100
	} else if method == "S" {
		realRate := rate * 0.01
		interest := (realRate / float64(12)) * float64(installmentInterval) * float64(installments) * capital
		instInterest := math.Round((interest/float64(installments))*100) / 100
		for i := 0; i < installments; i++ {
			initDate = initDate.AddDate(0, installmentInterval, 0)
			marketedSchedule[i] = Installment{
				Capital:         installmentCapital,
				Interest:        instInterest,
				DefaultInterest: 0,
				DueDate:         initDate.Format("2006-01-02"),
			}
		}

		capitalTotal := installmentCapital * float64(installments)
		capitalDiff := math.Round((capital-capitalTotal)*100) / 100
		marketedSchedule[installments-1].Capital = math.Round((marketedSchedule[installments-1].Capital+capitalDiff)*100) / 100
	} else if method == "R" {
		instInterest := (rate / (float64(12) / float64(installmentInterval))) * 0.01
		for i := 0; i < installments; i++ {
			initDate = initDate.AddDate(0, installmentInterval, 0)
			marketedSchedule[i] = Installment{
				Capital:         installmentCapital,
				Interest:        math.Round(((capital-(installmentCapital*float64(i)))*instInterest)*100) / 100,
				DefaultInterest: 0,
				DueDate:         initDate.Format("2006-01-02"),
			}
		}
	} else if method == "R2" {
		P := capital
		r := rate / float64(12) / 100
		n := installmentInterval * installments

		payment := math.Round((P*r*(math.Pow(1+r, float64(n))/(math.Pow(1+r, float64(n))-1)))*100) / 100

		// instInterest := (rate / (float64(12) / float64(installmentInterval))) * 0.01
		for i := 1; i <= installments; i++ {
			instInterest := float64(0)
			for j := (i-1)*installmentInterval + 1; j <= i*installmentInterval; j++ {
				rentalInterest := math.Round((((P*r)-payment)*math.Pow((r+1), (float64(j)-1))+payment)*100) / 100
				instInterest = instInterest + rentalInterest
			}

			initDate = initDate.AddDate(0, installmentInterval, 0)
			marketedSchedule[i-1] = Installment{
				Capital:         installmentCapital,
				Interest:        math.Round(instInterest*100) / 100,
				DefaultInterest: 0,
				DueDate:         initDate.Format("2006-01-02"),
			}
		}

		capitalTotal := installmentCapital * float64(installments)
		capitalDiff := math.Round((capital-capitalTotal)*100) / 100
		marketedSchedule[installments-1].Capital = math.Round((marketedSchedule[installments-1].Capital+capitalDiff)*100) / 100

		initDate, err = time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s 00:00:00", initiationDate))
		if initDate.Day() > 28 {
			initDate = initDate.AddDate(0, 0, -(initDate.Day() - 28))
		}
		capitalTotal = float64(0)
		for i := 1; i <= n; i++ {
			initDate = initDate.AddDate(0, 1, 0)
			rentalInterest := math.Round((((P*r)-payment)*math.Pow((r+1), (float64(i)-1))+payment)*100) / 100
			rentalCapital := math.Round((payment-rentalInterest)*100) / 100
			capitalTotal = capitalTotal + rentalCapital
			financialSchedule[i-1] = InstallmentSchedule{
				Capital:     rentalCapital,
				Interest:    rentalInterest,
				MonthlyDate: initDate.Format("2006-01-02"),
			}

			if i%installmentInterval == 0 {
				financialSchedule[i-1].MarketedInstallment = 1
				financialSchedule[i-1].MarketedCapital = marketedSchedule[i/installmentInterval-1].Capital
				financialSchedule[i-1].MarketedInterest = marketedSchedule[i/installmentInterval-1].Interest
				financialSchedule[i-1].MarketedDueDate = marketedSchedule[i/installmentInterval-1].DueDate
			} else {
				financialSchedule[i-1].MarketedDueDate = marketedSchedule[i/installmentInterval].DueDate
			}
		}
		capitalDiff = math.Round((P-capitalTotal)*100) / 100
		financialSchedule[n-1].Capital = math.Round((financialSchedule[n-1].Capital+capitalDiff)*100) / 100
	} else if method == "IRR" {
		P := capital
		r := rate / float64(12) / 100
		n := installmentInterval * installments

		payment := math.Round((P*r*(math.Pow(1+r, float64(n))/(math.Pow(1+r, float64(n))-1)))*100) / 100

		capitalTotal := float64(0)
		for i := 1; i <= installments; i++ {
			instInterest := float64(0)
			instCapital := float64(0)
			for j := (i-1)*installmentInterval + 1; j <= i*installmentInterval; j++ {
				rentalInterest := math.Round((((P*r)-payment)*math.Pow((r+1), (float64(j)-1))+payment)*100) / 100
				rentalCapital := math.Round((payment-rentalInterest)*100) / 100
				instInterest = instInterest + rentalInterest
				instCapital = instCapital + rentalCapital
			}

			initDate = initDate.AddDate(0, installmentInterval, 0)
			marketedSchedule[i-1] = Installment{
				Capital:         instCapital,
				Interest:        math.Round(instInterest*100) / 100,
				DefaultInterest: 0,
				DueDate:         initDate.Format("2006-01-02"),
			}
			capitalTotal = capitalTotal + instCapital
		}
		capitalDiff := math.Round((capital-capitalTotal)*100) / 100
		marketedSchedule[installments-1].Capital = math.Round((marketedSchedule[installments-1].Capital+capitalDiff)*100) / 100

		initDate, err = time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s 00:00:00", initiationDate))
		if initDate.Day() > 28 {
			initDate = initDate.AddDate(0, 0, -(initDate.Day() - 28))
		}
		capitalTotal = float64(0)
		for i := 1; i <= n; i++ {
			initDate = initDate.AddDate(0, 1, 0)
			rentalInterest := math.Round((((P*r)-payment)*math.Pow((r+1), (float64(i)-1))+payment)*100) / 100
			rentalCapital := math.Round((payment-rentalInterest)*100) / 100
			capitalTotal = capitalTotal + rentalCapital
			financialSchedule[i-1] = InstallmentSchedule{
				Capital:     rentalCapital,
				Interest:    rentalInterest,
				MonthlyDate: initDate.Format("2006-01-02"),
			}

			if i%installmentInterval == 0 {
				financialSchedule[i-1].MarketedInstallment = 1
				financialSchedule[i-1].MarketedCapital = marketedSchedule[i/installmentInterval-1].Capital
				financialSchedule[i-1].MarketedInterest = marketedSchedule[i/installmentInterval-1].Interest
				financialSchedule[i-1].MarketedDueDate = marketedSchedule[i/installmentInterval-1].DueDate
			} else {
				financialSchedule[i-1].MarketedDueDate = marketedSchedule[i/installmentInterval].DueDate
			}
		}
		capitalDiff = math.Round((P-capitalTotal)*100) / 100
		financialSchedule[n-1].Capital = math.Round((financialSchedule[n-1].Capital+capitalDiff)*100) / 100
	} else if method == "SM" {
		realRate := math.Round((rate*0.01)*100) / 100

		interest := (realRate * (float64(installments) * float64(7) / float64(365))) * capital
		instInterest := math.Round((interest/float64(installments))*100) / 100

		for i := 0; i < installments; i++ {
			initDate = initDate.AddDate(0, 0, 7)

			if initDate.Month() == 4 && initDate.Day() >= 10 && initDate.Day() <= 16 {
				initDate = initDate.AddDate(0, 0, 7)
			}

			if initDate.Month() == 12 && initDate.Day() >= 23 && initDate.Day() <= 29 {
				initDate = initDate.AddDate(0, 0, 7)
			}

			marketedSchedule[i] = Installment{
				Capital:         installmentCapital,
				Interest:        instInterest,
				DefaultInterest: 0,
				DueDate:         initDate.Format("2006-01-02"),
			}
		}

		capitalTotal := installmentCapital * float64(installments)
		capitalDiff := math.Round((capital-capitalTotal)*100) / 100
		marketedSchedule[installments-1].Capital = math.Round((marketedSchedule[installments-1].Capital+capitalDiff)*100) / 100

		for i := 0; i < len(financialSchedule); i++ {
			financialSchedule[i].Capital = marketedSchedule[i].Capital
			financialSchedule[i].Interest = marketedSchedule[i].Interest
			financialSchedule[i].MonthlyDate = marketedSchedule[i].DueDate
			financialSchedule[i].MarketedInstallment = 1
			financialSchedule[i].MarketedCapital = marketedSchedule[i].Capital
			financialSchedule[i].MarketedInterest = marketedSchedule[i].Interest
			financialSchedule[i].MarketedDueDate = marketedSchedule[i].DueDate
		}
	}

	return marketedSchedule, financialSchedule, nil
}
