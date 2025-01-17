package co

import (
	"errors"
	"fmt"
	"strconv"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/common"
	"github.com/invopop/gobl/tax"
)

var (
	nitMultipliers = []int{3, 7, 13, 17, 19, 23, 29, 37, 41, 43, 47, 53, 59, 67, 71}
)

// validateTaxIdentity checks to ensure the NIT code looks okay.
func validateTaxIdentity(tID *tax.Identity) error {
	return validation.ValidateStruct(tID,
		validation.Field(&tID.Code, validation.Required, validation.By(validateTaxCode)),
		validation.Field(&tID.Zone, validation.Required, isValidZoneCode),
	)
}

// normalizeTaxIdentity will remove any whitespace or separation characters from
// the tax code.
func normalizeTaxIdentity(tID *tax.Identity) error {
	if err := common.NormalizeTaxIdentity(tID); err != nil {
		return err
	}
	return nil
}

func normalizePartyWithTaxIdentity(p *org.Party) error {
	// override the party's locality and region using the tax identity zone data.
	tID := p.TaxID
	if tID != nil && tID.Zone != "" {
		z := zoneForCode(tID.Zone)
		if z != nil {
			if len(p.Addresses) == 0 {
				return nil
			}
			a := p.Addresses[0]
			a.Locality = z.Locality.String(i18n.ES)
			a.Region = z.Region.String(i18n.ES)
		}
	}
	return nil
}

var isValidZoneCode = validation.In(validZoneCodes()...)

func validZoneCodes() []interface{} {
	ls := make([]interface{}, len(zones))
	for i, v := range zones {
		ls[i] = v.Code
	}
	return ls
}

func validateTaxCode(value interface{}) error {
	code, ok := value.(cbc.Code)
	if !ok {
		return nil
	}
	if code == "" {
		return nil
	}
	for _, v := range code {
		x := v - 48
		if x < 0 || x > 9 {
			return errors.New("contains invalid characters")
		}
	}
	l := len(code)
	if l > 10 {
		return errors.New("too long")
	}
	if l < 9 {
		return errors.New("too short")
	}

	return validateDigits(code[0:l-1], code[l-1:l])
}

func validateDigits(code, check cbc.Code) error {
	ck, err := strconv.Atoi(string(check))
	if err != nil {
		return fmt.Errorf("invalid check: %w", err)
	}

	sum := 0
	l := len(code)
	for i, v := range code {
		// 48 == ASCII "0"
		sum += int(v-48) * nitMultipliers[l-i-1]
	}
	sum = sum % 11
	if sum >= 2 {
		sum = 11 - sum
	}

	if sum != ck {
		return errors.New("checksum mismatch")
	}

	return nil
}
