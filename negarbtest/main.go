package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/auth/credentials/externalaccount"
	"cloud.google.com/go/auth/credentials/idtoken"
	"cloud.google.com/go/auth/httptransport"
	"cloud.google.com/go/storage"

	"google.golang.org/api/option"
)

// // java: impersonatedCredentials
// func testServiceAccount_detect_impersonate() {
// 	ctx := context.Background()
// 	// Replace with the actual service account you want to impersonate
// 	impersonatedServiceAccount := "tb-us-central1@iamcredentials-prober.iam.gserviceaccount.com"

// 	// Use credentials.Default() to get credentials for the current environment
// 	creds, _ := credentials.DetectDefault(&credentials.DetectOptions{
// 		Scopes: []string{
// 			"https://www.googleapis.com/auth/cloud-platform",
// 		}})

// 	// Create a new context with the impersonated service account
// 	md := metadata.Pairs("x-goog-impersonate-service-account", impersonatedServiceAccount)
// 	ctx = metadata.NewOutgoingContext(ctx, md)

// 	token, err := creds.Token(ctx)
// 	if err != nil {
// 		fmt.Println("Error getting token:", err)
// 		return
// 	}

// 	// println("Successfully obtained token with trust boundary: " + token.TrustBoundaryData.EncodedLocations())
// }

func main() {
	// workforcePoolExample()
	// testServiceAccount_client_side_cab()
	testServiceAccount_detect_default()
	// testServiceAccount_detect_impersonate()
	// x509Testing()
}

func workforcePoolExample() {
	projectId := "wf-pools-testing"
	// url := "https://iam.googleapis.com/v1/projects/" + projectId + "/serviceAccounts"

	opts := externalaccount.Options{
		Audience:                 "//iam.googleapis.com/locations/global/workforcePools/wf-pools-testing-sdk/providers/okta-oidc-provider",
		SubjectTokenType:         "urn:ietf:params:oauth:token-type:id-token",
		WorkforcePoolUserProject: projectId,
		SubjectTokenProvider:     nil,
	}

	creds, err := externalaccount.NewCredentials(&opts)
	if err != nil {
		fmt.Println("Error creating credentials:", err)
		return
	}

	fmt.Println("token created with %v", creds.TokenProvider)
}

// func x509Testing() {
// 	ctx := context.Background()
// 	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
// 		Scopes: []string{
// 			"https://www.googleapis.com/auth/cloud-platform",
// 		}})
// 	if err != nil {
// 		fmt.Println("Error detecting credentials:", err)
// 		return
// 	}

// 	if credsJSON := creds.JSON(); credsJSON != nil {
// 		var f struct {
// 			ClientEmail string `json:"client_email"`
// 			Type        string `json:"type"`
// 		}
// 		if err := json.Unmarshal(credsJSON, &f); err == nil {
// 			fmt.Printf("Detected credential type from file: %s\n", f.Type)
// 			if f.ClientEmail != "" {
// 				fmt.Printf("Service Account Email: %s\n", f.ClientEmail)
// 			}
// 		}
// 	} else {
// 		fmt.Println("Detected credentials from environment (e.g., GCE metadata server).")
// 	}

// 	token, err := creds.Token(ctx)
// 	if err != nil {
// 		fmt.Println("Error getting token:", err)
// 		return
// 	}

// 	println(token)
// 	// location := token.TrustBoundaryData.Locations()
// 	// fmt.Println(location)
// }

// headerInspectionTransport is a custom http.RoundTripper to print request headers.
type headerInspectionTransport struct {
	base http.RoundTripper
}

func (t *headerInspectionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Println("\n--- Inspecting Outgoing Request Headers ---")
	for name, values := range req.Header {
		for _, value := range values {
			// Print the Authorization header value's prefix for brevity.
			if name == "Authorization" && len(value) > 20 {
				fmt.Printf("%s: %s...\n", name, value)
			} else {
				fmt.Printf("%s: %s\n", name, value)
			}
		}
	}
	fmt.Println("-----------------------------------------")

	baseTransport := t.base
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	return baseTransport.RoundTrip(req)
}

func testServiceAccount_detect_default() {
	ctx := context.Background()
	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
		Scopes: []string{
			"https://www.googleapis.com/auth/cloud-platform",
		}})
	if err != nil {
		fmt.Println("Error detecting credentials:", err)
		return
	}

	if credsJSON := creds.JSON(); credsJSON != nil {
		var f struct {
			ClientEmail string `json:"client_email"`
			Type        string `json:"type"`
		}
		if err := json.Unmarshal(credsJSON, &f); err == nil {
			fmt.Printf("Detected credential type from file: %s\n", f.Type)
			if f.ClientEmail != "" {
				fmt.Printf("Service Account Email: %s\n", f.ClientEmail)
			}
		}
	} else {
		fmt.Println("Detected credentials from environment (e.g., GCE metadata server).")
	}

	// The auth library handles token fetching internally when making an API call.
	// We can still fetch one manually to see the initial data.
	token, err := creds.Token(ctx)
	if err != nil {
		fmt.Println("Error getting token:", err)
		return
	}

	println("Successfully obtained token with EncodedLocations: " + token.TrustBoundaryData.EncodedLocations)
	location := token.TrustBoundaryData.Locations
	fmt.Println("Locations from initial token fetch:", location)
	// fmt.Println("Locations: " + location[len(location)-1])

	// Create the base transport that will inspect the final request headers.
	inspectionTransport := &headerInspectionTransport{
		base: http.DefaultTransport,
	}

	// Use httptransport.NewClient to construct a client with the modern,
	// correct transport chain that understands trust boundaries. We pass our
	// inspection transport as the base to ensure it's part of the chain.
	httpClient, err := httptransport.NewClient(&httptransport.Options{
		Credentials:      creds,
		BaseRoundTripper: inspectionTransport,
	})
	if err != nil {
		log.Fatalf("Failed to create httptransport.NewClient: %v", err)
	}

	// Create the storage client using the fully-configured HTTP client.
	storageService, err := storage.NewClient(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		log.Fatalf("Failed to create storage client: %v", err)
	}
	defer storageService.Close()

	// Get project ID from credentials to make a real API call.
	projectID, err := creds.ProjectID(ctx)
	if err != nil || projectID == "" {
		// Fallback project ID if not found in credentials.
		// You may need to change this to a project you have access to.
		projectID = "gcs-trust-boundary-prober"
		fmt.Printf("Could not determine project ID from credentials, using fallback: %s\n", projectID)
	}

	fmt.Printf("\nMaking a Storage API call to list buckets for project: %s\n", projectID)
	// Make a simple API call to trigger the RoundTripper and see the headers.
	it := storageService.Buckets(ctx, projectID)
	_, err = it.Next()
	if err != nil {
		// An error here is okay for this test (e.g., no buckets, or permissions issue).
		// The main goal is to inspect the headers, which happens before the response is processed.
		log.Printf("Storage API call finished. Error (if any) is expected: %v\n", err)
	} else {
		log.Println("Storage API call finished successfully.")
	}
}

func testServiceAccount_client_side_cab() {
	println("hey :)")
	ctx := context.Background()
	opts := &idtoken.Options{
		Audience:        "https://storage.googleapis.com/",
		CredentialsFile: "/Users/negarb/workplace/java/client-side-cab.json",
	}

	ts, err := idtoken.NewCredentials(opts)

	storageService, err := storage.NewClient(ctx, option.WithAuthCredentials(ts))
	if err != nil {
		fmt.Println("Error creating storage service:", err)
		return
	}

	defer storageService.Close()

	// projectID := "your-gcp-project-id"
	bucketName := "client-side-cab"
	bucket := storageService.Bucket(bucketName)

	bucketAttrs, err := bucket.Attrs(ctx)
	if err != nil {
		log.Printf("Error getting bucket attributes for %s: %v", bucketName, err)
		return
	}

	// Print bucket information.
	fmt.Printf("Bucket Name: %s\n", bucketAttrs.Name)
	fmt.Printf("Bucket Created: %s\n", bucketAttrs.Created)
	fmt.Printf("Bucket Location: %s\n", bucketAttrs.Location)

}

// func default_workforce() {
// 	const projectId = "wf-pools-testing"
// 	const url = "https://iam.googleapis.com/v1/projects/" + projectId + "/serviceAccounts"

// 	ctx := context.Background()
// 	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
// 		Scopes: []string{
// 			"https://www.googleapis.com/auth/cloud-platform",
// 		}})

// 	if err != nil {
// 		fmt.Println("Error detecting credentials:", err)
// 		return
// 	}

// 	token, err := creds.Token(ctx)
// 	if err != nil {
// 		fmt.Println("Error getting token:", err)
// 		return
// 	}

// 	// println("Successfully pbtained token with trust boundary: " + token.TrustBoundaryData.EncodedLocations)

// 	// client, err := credentials.DetectDefault(context.Background(), scopes...)
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }

// 	// resp, err := client.Get(url)
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }
// 	// defer resp.Body.Close()

// 	// io.Copy(log.Writer(), resp.Body)
// }
