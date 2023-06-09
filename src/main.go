package main

import (
	"errors"
	"flag"
	"io/ioutil"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"
	_ "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	//	"net/http/httputil"
	"strconv"
	"sync"
)

// ServerParameters : we need to enable a TLS endpoint
// Let's take some parameters where we can set the path to the TLS certificate and port number to run on.
type ServerParameters struct {
	port     int    // webhook server port
	certFile string // path to the x509 certificate for https
	keyFile  string // path to the x509 private key matching `CertFile`
}

// To perform a simple mutation on the object before the Kubernetes API sees the object, we can apply a patch to the operation. RFC6902
type patchOperation struct {
	Op    string      `json:"op"`   // Operation
	Path  string      `json:"path"` // Path
	Value interface{} `json:"value,omitempty"`
}

// Config To perform patching to Pod definition
type Config struct {
	Containers []corev1.Container `yaml:"containers"`
	Volumes    []corev1.Volume    `yaml:"volumes"`
}

var (
	universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	k8sConfig             *rest.Config
	k8sClientSet          *kubernetes.Clientset
	serverParameters      ServerParameters
	m                     sync.Mutex
)

func main() {
	flag.IntVar(&serverParameters.port, "port", 8443, "Webhook server port.")
	flag.StringVar(&serverParameters.certFile, "tlsCertFile", "/etc/webhook/certs/tls.crt", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&serverParameters.keyFile, "tlsKeyFile", "/etc/webhook/certs/tls.key", "File containing the x509 private key to --tlsCertFile.")
	flag.Parse()

	// Creating Client Set.
	k8sClientSet = createClientSet()

	logger.Info("Starting webhook server...")
	http.HandleFunc("/", HandleRoot)
	http.HandleFunc("/mutate", HandleMutate)
	logger.Fatal(http.ListenAndServeTLS(":"+strconv.Itoa(serverParameters.port), serverParameters.certFile, serverParameters.keyFile, nil))
}

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("HandleRoot!"))
	if err != nil {
		return
	}
}

func getAdmissionReviewRequest(w http.ResponseWriter, r *http.Request) admissionv1.AdmissionReview {
	//requestDump, _ := httputil.DumpRequest(r, true)
	//fmt.Printf("Request:\n%s\n", requestDump)

	// Grabbing the http body received on webhook.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Panic("Error reading webhook request: ", err.Error())
	}

	// Required to pass to universal decoder.
	// v1beta1 also needs to be added to webhook.yaml
	var admissionReviewReq admissionv1.AdmissionReview

	//logger.Info("deserializing admission review request")
	if _, _, err := universalDeserializer.Decode(body, nil, &admissionReviewReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logger.Errorf("could not deserialize request: %v", err)
	} else if admissionReviewReq.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = errors.New("malformed admission review: request is nil")
	}
	return admissionReviewReq
}

func HandleMutate(w http.ResponseWriter, r *http.Request) {
	// func getAdmissionReviewRequest, grab body from request, define AdmissionReview
	// and use universalDeserializer to decode body to admissionReviewReq
	admissionReviewReq := getAdmissionReviewRequest(w, r)
	//uniqueId := string(admissionReviewReq.Request.UID)

	// Debug statement to verify if universalDeserializer worked
	logger.Infof("Type: %v, Event: %v, Id: %v", admissionReviewReq.Request.Kind, admissionReviewReq.Request.Operation, admissionReviewReq.Request.UID)

	// We now need to capture Pod object from the admission request
	var pod corev1.Pod
	err := json.Unmarshal(admissionReviewReq.Request.Object.Raw, &pod)
	if err != nil {
		logger.Errorf("could not unmarshal pod on admission request: %v", err)
	}

	//podBytes, _ := json.Marshal(&pod)
	//fmt.Printf("old document: %s\n", podBytes)

	// To perform a mutation on the object before the Kubernetes API sees the object, we can apply a patch to the operation
	// Add Labels
	//var sideCarConfig *Config
	//sideCarConfig = getNginxSideCarConfig(uniqueId)

	patches, _ := createPatch(pod)

	// Once you have completed all patching, convert the patches to byte slice:
	patchBytes, err := json.Marshal(patches)
	//fmt.Printf("patch document: %s\n", podBytes)

	if err != nil {
		logger.Errorf("could not marshal JSON patch: %v", err)
	}

	//logger.Infof("patchBytes: %s", string(patchBytes))

	// Add patchBytes to the admission response
	admissionReviewResponse := admissionv1.AdmissionReview{
		Response: &admissionv1.AdmissionResponse{
			UID:     admissionReviewReq.Request.UID,
			Allowed: true,
		},
	}
	admissionReviewResponse.Response.Patch = patchBytes

	// Submit the response
	bytes, err := json.Marshal(&admissionReviewResponse)
	if err != nil {
		logger.Errorf("marshaling response: %v", err)
	}

	_, err = w.Write(bytes)
	if err != nil {
		return
	}

}
