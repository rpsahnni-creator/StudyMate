package auth

import "strings"

// UserFacingOTPDeliveryError maps provider errors to short signup messages.
func UserFacingOTPDeliveryError(err error) string {
	if err == nil {
		return ""
	}
	lower := strings.ToLower(err.Error())

	if strings.Contains(lower, "only send testing emails to your own email address") ||
		(strings.Contains(lower, "resend error 403") && strings.Contains(lower, "verify a domain")) {
		return "Resend test mode: OTP sirf aapke Resend account wale Gmail par ja sakta hai. " +
			"Kisi bhi student email ke liye resend.com/domains par domain verify karein (e.g. kijitechnology.com)."
	}

	return err.Error()
}
