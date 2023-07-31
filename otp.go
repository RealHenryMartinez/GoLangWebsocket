// Purpose of file: One time password handlers

package main

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// One time password
type OTP struct {
	Key     string    // The password itself
	Created time.Time // Time password was created
}

// Store the paswords
type RetentionMap map[string]OTP

// Making a copy of OTP
// ctx - typically used to manage cancellations and timeouts of a function
// retentionPeriod - how long before the password will expire
func NewRetentionMap(ctx context.Context, retentionPeriod time.Duration) RetentionMap {
	rm := make(RetentionMap) // Create a map from the retention map type

	go rm.Retention(ctx, retentionPeriod)

	return rm
}

// Adding a one time pass to the retention map
// Funtion not affecting the original rm referenced
func (rm RetentionMap) NewOTP() OTP {
	o := OTP{
		Key:     uuid.NewString(),
		Created: time.Now(),
	}

	// make the key being the password and put the value inside the retention map
	rm[o.Key] = o
	return o // Return the OTP
}

// VerifyOTP will make sure a OTP exists
// and return true if so
// It will also delete the key so it cant be reused
func (rm RetentionMap) VerifyOTP(otp string) bool {
	// Verify the existence of the OTP
	if _, ok := rm[otp]; !ok {
		// OTP does not exist
		return false
	}

	delete(rm, otp) // Delete the OTP from the retention map
	return true
}

// Removing old OTP's.
// The method compares the creation time of each OTP with the current time and removes any OTPs that have exceeded the retentionPeriod
func (rm RetentionMap) Retention(ctx context.Context, retentionPeriod time.Duration) {
	ticker := time.NewTicker(400 * time.Millisecond)
	for {
		select {
			case <-ticker.C:
				// Loop through all the OTP (index could be ignored) 
				for _, otp := range rm {
					// Add the time created to the time created property of the OTP
					if otp.Created.Add(retentionPeriod).Before(time.Now()) {
						// Updating the creation time if it is not already
						delete(rm, otp.Key)
					}
				}
			// Check if this goroutine channel is already finished running or cancelled 
			case <- ctx.Done():
				return // Close the goroutine channel
		}
	}
}