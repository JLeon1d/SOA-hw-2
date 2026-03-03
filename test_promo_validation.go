//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"time"
)

type PromoCode struct {
	Active      bool
	CurrentUses int
	MaxUses     int
	ValidFrom   time.Time
	ValidUntil  time.Time
}

func (p *PromoCode) IsValid() bool {
	now := time.Now()
	return p.Active &&
		p.CurrentUses < p.MaxUses &&
		!now.Before(p.ValidFrom) &&
		!now.After(p.ValidUntil)
}

func main() {
	// Simulate test scenario
	now := time.Now()
	promo := &PromoCode{
		Active:      true,
		CurrentUses: 0,
		MaxUses:     100,
		ValidFrom:   now,
		ValidUntil:  now.Add(30 * 24 * time.Hour),
	}

	fmt.Printf("Promo created at: %v\n", now)
	fmt.Printf("ValidFrom: %v\n", promo.ValidFrom)
	fmt.Printf("ValidUntil: %v\n", promo.ValidUntil)
	fmt.Printf("Active: %v\n", promo.Active)
	fmt.Printf("CurrentUses: %d, MaxUses: %d\n", promo.CurrentUses, promo.MaxUses)

	// Check immediately
	checkTime := time.Now()
	fmt.Printf("\nChecking at: %v\n", checkTime)
	fmt.Printf("!checkTime.Before(ValidFrom): %v\n", !checkTime.Before(promo.ValidFrom))
	fmt.Printf("!checkTime.After(ValidUntil): %v\n", !checkTime.After(promo.ValidUntil))
	fmt.Printf("IsValid(): %v\n", promo.IsValid())

	// Check with exact ValidFrom time
	fmt.Printf("\nChecking at exact ValidFrom time:\n")
	fmt.Printf("!ValidFrom.Before(ValidFrom): %v\n", !promo.ValidFrom.Before(promo.ValidFrom))
	fmt.Printf("!ValidFrom.After(ValidUntil): %v\n", !promo.ValidFrom.After(promo.ValidUntil))

	// Check with exact ValidUntil time
	fmt.Printf("\nChecking at exact ValidUntil time:\n")
	fmt.Printf("!ValidUntil.Before(ValidFrom): %v\n", !promo.ValidUntil.Before(promo.ValidFrom))
	fmt.Printf("!ValidUntil.After(ValidUntil): %v\n", !promo.ValidUntil.After(promo.ValidUntil))
}
