// Copyright © 2019 VMware
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dag

import (
	"testing"

	ingressroutev1 "github.com/projectcontour/contour/apis/contour/v1beta1"
	projcontour "github.com/projectcontour/contour/apis/projectcontour/v1"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKubernetesCacheInsert(t *testing.T) {
	tests := map[string]struct {
		pre  []interface{}
		obj  interface{}
		want bool
	}{
		"insert secret": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
			},
			want: false,
		},
		"insert secret w/ blank ca.crt": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: map[string][]byte{
					"ca.crt":            []byte(""),
					v1.TLSCertKey:       []byte(CERTIFICATE),
					v1.TLSPrivateKeyKey: []byte(RSA_PRIVATE_KEY),
				},
			},
			want: true,
		},
		"insert CA secret w/ explanatory text": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{
					"ca.crt": []byte(CERTIFICATE_WITH_TEXT),
				},
			},
			want: true,
		},
		"insert CA bundle secret w/ non-PEM data": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeOpaque,
				Data: caBundleData(CERTIFICATE, CERTIFICATE, CERTIFICATE, CERTIFICATE),
			},
			want: true,
		},
		"insert CA bundle secret w/ non-PEM data and no certificates": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeOpaque,
				Data: caBundleData(),
			},
			want: false,
		},

		"insert secret referenced by ingress": {
			pre: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "www",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							SecretName: "secret",
						}},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
			},
			want: true,
		},
		"insert secret referenced by ingress with multiple pem blocks": {
			pre: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "www",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							SecretName: "secret",
						}},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(EC_CERTIFICATE, EC_PRIVATE_KEY),
			},
			want: true,
		},
		"insert secret w/ wrong type referenced by ingress": {
			pre: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "www",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							SecretName: "secret",
						}},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: "banana",
			},
			want: false,
		},
		"insert secret referenced by ingress via tls delegation": {
			pre: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "www",
						Namespace: "extra",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							SecretName: "default/secret",
						}},
					},
				},
				&ingressroutev1.TLSCertificateDelegation{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "delegation",
						Namespace: "default",
					},
					Spec: ingressroutev1.TLSCertificateDelegationSpec{
						Delegations: []ingressroutev1.CertificateDelegation{{
							SecretName: "secret",
							TargetNamespaces: []string{
								"extra",
							},
						}},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
			},
			want: true,
		},
		"insert secret referenced by ingress via wildcard tls delegation": {
			pre: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "www",
						Namespace: "extra",
					},
					Spec: v1beta1.IngressSpec{
						TLS: []v1beta1.IngressTLS{{
							SecretName: "default/secret",
						}},
					},
				},

				&ingressroutev1.TLSCertificateDelegation{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "delegation",
						Namespace: "default",
					},
					Spec: ingressroutev1.TLSCertificateDelegationSpec{
						Delegations: []ingressroutev1.CertificateDelegation{{
							SecretName: "secret",
							TargetNamespaces: []string{
								"*",
							},
						}},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
			},
			want: true,
		},
		"insert secret referenced by ingressroute": {
			pre: []interface{}{
				&ingressroutev1.IngressRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: ingressroutev1.IngressRouteSpec{
						VirtualHost: &projcontour.VirtualHost{
							TLS: &projcontour.TLS{
								SecretName: "secret",
							},
						},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
			},
			want: true,
		},
		"insert secret referenced by ingressroute via tls delegation": {
			pre: []interface{}{
				&ingressroutev1.IngressRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "extra",
					},
					Spec: ingressroutev1.IngressRouteSpec{
						VirtualHost: &projcontour.VirtualHost{
							TLS: &projcontour.TLS{
								SecretName: "default/secret",
							},
						},
					},
				},
				&ingressroutev1.TLSCertificateDelegation{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "delegation",
						Namespace: "default",
					},
					Spec: ingressroutev1.TLSCertificateDelegationSpec{
						Delegations: []ingressroutev1.CertificateDelegation{{
							SecretName: "secret",
							TargetNamespaces: []string{
								"extra",
							},
						}},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
			},
			want: true,
		},
		"insert secret referenced by ingressroute via wildcard tls delegation": {
			pre: []interface{}{
				&ingressroutev1.IngressRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "extra",
					},
					Spec: ingressroutev1.IngressRouteSpec{
						VirtualHost: &projcontour.VirtualHost{
							TLS: &projcontour.TLS{
								SecretName: "default/secret",
							},
						},
					},
				},
				&ingressroutev1.TLSCertificateDelegation{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "delegation",
						Namespace: "default",
					},
					Spec: ingressroutev1.TLSCertificateDelegationSpec{
						Delegations: []ingressroutev1.CertificateDelegation{{
							SecretName: "secret",
							TargetNamespaces: []string{
								"*",
							},
						}},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
			},
			want: true,
		},

		"insert secret referenced by httpproxy": {
			pre: []interface{}{
				&projcontour.HTTPProxy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "default",
					},
					Spec: projcontour.HTTPProxySpec{
						VirtualHost: &projcontour.VirtualHost{
							TLS: &projcontour.TLS{
								SecretName: "secret",
							},
						},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
			},
			want: true,
		},
		"insert secret referenced by httpproxy via tls delegation": {
			pre: []interface{}{
				&projcontour.HTTPProxy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "extra",
					},
					Spec: projcontour.HTTPProxySpec{
						VirtualHost: &projcontour.VirtualHost{
							TLS: &projcontour.TLS{
								SecretName: "default/secret",
							},
						},
					},
				},
				&projcontour.TLSCertificateDelegation{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "delegation",
						Namespace: "default",
					},
					Spec: projcontour.TLSCertificateDelegationSpec{
						Delegations: []projcontour.CertificateDelegation{{
							SecretName: "secret",
							TargetNamespaces: []string{
								"extra",
							},
						}},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
			},
			want: true,
		},
		"insert secret referenced by httpproxy via wildcard tls delegation": {
			pre: []interface{}{
				&projcontour.HTTPProxy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: "extra",
					},
					Spec: projcontour.HTTPProxySpec{
						VirtualHost: &projcontour.VirtualHost{
							TLS: &projcontour.TLS{
								SecretName: "default/secret",
							},
						},
					},
				},
				&projcontour.TLSCertificateDelegation{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "delegation",
						Namespace: "default",
					},
					Spec: projcontour.TLSCertificateDelegationSpec{
						Delegations: []projcontour.CertificateDelegation{{
							SecretName: "secret",
							TargetNamespaces: []string{
								"*",
							},
						}},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: secretdata(CERTIFICATE, RSA_PRIVATE_KEY),
			},
			want: true,
		},
		"insert certificate secret": {
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ca",
					Namespace: "default",
				},
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{
					"ca.crt": []byte(CERTIFICATE),
				},
			},
			// TODO(dfc) this should be false because the CA secret is
			// not referenced, but computing its reference duplicates the
			// work done rebuilding the dag so for the moment assume that
			// any CA secret causes a rebuild.
			want: true,
		},
		"insert certificate secret referenced by ingressroute": {
			pre: []interface{}{
				&ingressroutev1.IngressRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-com",
						Namespace: "default",
					},
					Spec: ingressroutev1.IngressRouteSpec{
						VirtualHost: &projcontour.VirtualHost{
							Fqdn: "example.com",
						},
						Routes: []ingressroutev1.Route{{
							Match: "/",
							Services: []ingressroutev1.Service{{
								Name: "kuard",
								Port: 8080,
								UpstreamValidation: &projcontour.UpstreamValidation{
									CACertificate: "ca",
									SubjectName:   "example.com",
								},
							}},
						}},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ca",
					Namespace: "default",
				},
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{
					"ca.crt": []byte(CERTIFICATE),
				},
			},
			want: true,
		},
		"insert certificate secret referenced by httpproxy": {
			pre: []interface{}{
				&projcontour.HTTPProxy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-com",
						Namespace: "default",
					},
					Spec: projcontour.HTTPProxySpec{
						VirtualHost: &projcontour.VirtualHost{
							Fqdn: "example.com",
						},
						Routes: []projcontour.Route{{
							Conditions: []projcontour.Condition{{
								Prefix: "/",
							}},
							Services: []projcontour.Service{{
								Name: "kuard",
								Port: 8080,
								UpstreamValidation: &projcontour.UpstreamValidation{
									CACertificate: "ca",
									SubjectName:   "example.com",
								},
							}},
						}},
					},
				},
			},
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ca",
					Namespace: "default",
				},
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{
					"ca.crt": []byte(CERTIFICATE),
				},
			},
			want: true,
		},
		"insert ingress empty ingress class": {
			obj: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "incorrect",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert ingress incorrect kubernetes.io/ingress.class": {
			obj: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "incorrect",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			},
			want: false,
		},
		"insert ingress incorrect contour.heptio.com/ingress.class": {
			obj: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "incorrect",
					Namespace: "default",
					Annotations: map[string]string{
						"contour.heptio.com/ingress.class": "nginx",
					},
				},
			},
			want: false,
		},
		"insert ingress explicit kubernetes.io/ingress.class": {
			obj: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "incorrect",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": new(KubernetesCache).ingressClass(),
					},
				},
			},
			want: true,
		},
		"insert ingress explicit contour.heptio.com/ingress.class": {
			obj: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "incorrect",
					Namespace: "default",
					Annotations: map[string]string{
						"contour.heptio.com/ingress.class": new(KubernetesCache).ingressClass(),
					},
				},
			},
			want: true,
		},
		"insert ingressroute empty ingress annotation": {
			obj: &ingressroutev1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kuard",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert ingressroute incorrect contour.heptio.com/ingress.class": {
			obj: &ingressroutev1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple",
					Namespace: "default",
					Annotations: map[string]string{
						"contour.heptio.com/ingress.class": "nginx",
					},
				},
			},
			want: false,
		},
		"insert ingressroute incorrect kubernetes.io/ingress.class": {
			obj: &ingressroutev1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			},
			want: false,
		},
		"insert ingressroute: explicit contour.heptio.com/ingress.class": {
			obj: &ingressroutev1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kuard",
					Namespace: "default",
					Annotations: map[string]string{
						"contour.heptio.com/ingress.class": new(KubernetesCache).ingressClass(),
					},
				},
			},
			want: true,
		},
		"insert ingressroute explicit kubernetes.io/ingress.class": {
			obj: &ingressroutev1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kuard",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": new(KubernetesCache).ingressClass(),
					},
				},
			},
			want: true,
		},
		"insert httpproxy empty ingress annotation": {
			obj: &projcontour.HTTPProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kuard",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert httpproxy incorrect contour.heptio.com/ingress.class": {
			obj: &projcontour.HTTPProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple",
					Namespace: "default",
					Annotations: map[string]string{
						"contour.heptio.com/ingress.class": "nginx",
					},
				},
			},
			want: false,
		},
		"insert httpproxy incorrect kubernetes.io/ingress.class": {
			obj: &projcontour.HTTPProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			},
			want: false,
		},
		"insert httpproxy: explicit contour.heptio.com/ingress.class": {
			obj: &projcontour.HTTPProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kuard",
					Namespace: "default",
					Annotations: map[string]string{
						"contour.heptio.com/ingress.class": new(KubernetesCache).ingressClass(),
					},
				},
			},
			want: true,
		},
		"insert httpproxy explicit kubernetes.io/ingress.class": {
			obj: &projcontour.HTTPProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kuard",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": new(KubernetesCache).ingressClass(),
					},
				},
			},
			want: true,
		},
		"insert tls contour/v1beta1.certificate delegation": {
			obj: &ingressroutev1.TLSCertificateDelegation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "delegate",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert tls projcontour/v1.certificatedelegation": {
			obj: &projcontour.TLSCertificateDelegation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "delegate",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert httpproxy": {
			obj: &projcontour.HTTPProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "httpproxy",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert unknown": {
			obj:  "not an object",
			want: false,
		},
		"insert service": {
			obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "default",
				},
			},
			want: false,
		},
		"insert service referenced by ingress backend": {
			pre: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "www",
						Namespace: "default",
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "service",
						},
					},
				},
			},
			obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert service in different namespace": {
			pre: []interface{}{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "www",
						Namespace: "kube-system",
					},
					Spec: v1beta1.IngressSpec{
						Backend: &v1beta1.IngressBackend{
							ServiceName: "service",
						},
					},
				},
			},
			obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "default",
				},
			},
			want: false,
		},
		"insert service referenced by ingressroute": {
			pre: []interface{}{
				&ingressroutev1.IngressRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: ingressroutev1.IngressRouteSpec{
						Routes: []ingressroutev1.Route{{
							Services: []ingressroutev1.Service{{
								Name: "service",
							}},
						}},
					},
				},
			},
			obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert service referenced by ingressroute tcpproxy": {
			pre: []interface{}{
				&ingressroutev1.IngressRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: ingressroutev1.IngressRouteSpec{
						TCPProxy: &ingressroutev1.TCPProxy{
							Services: []ingressroutev1.Service{{
								Name: "service",
							}},
						},
					},
				},
			},
			obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert service referenced by httpproxy": {
			pre: []interface{}{
				&projcontour.HTTPProxy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: projcontour.HTTPProxySpec{
						Routes: []projcontour.Route{{
							Services: []projcontour.Service{{
								Name: "service",
							}},
						}},
					},
				},
			},
			obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "default",
				},
			},
			want: true,
		},
		"insert service referenced by httpproxy tcpproxy": {
			pre: []interface{}{
				&projcontour.HTTPProxy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
					Spec: projcontour.HTTPProxySpec{
						TCPProxy: &projcontour.TCPProxy{
							Services: []projcontour.Service{{
								Name: "service",
							}},
						},
					},
				},
			},
			obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "default",
				},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			builder := Builder{
				Source: KubernetesCache{
					FieldLogger: testLogger(t),	
				},
			}
			for _, p := range tc.pre {
				builder.CacheInsert(p)
			}
			got := builder.CacheInsert(tc.obj)
			if tc.want != got {
				t.Fatalf("Insert(%v): expected %v, got %v", tc.obj, tc.want, got)
			}
		})
	}
}

func TestKubernetesCacheRemove(t *testing.T) {

	builder := func(objs ...interface{}) *Builder {
		builder := Builder{
			Source: KubernetesCache{
				FieldLogger: testLogger(t),	
			},
		}
		for _, o := range objs {
			builder.CacheInsert(o)
		}
		return &builder
	}

	tests := map[string]struct {
		builder *Builder
		obj   interface{}
		want  bool
	}{
		"remove secret": {
			builder: builder(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
				Data: map[string][]byte{
					v1.TLSCertKey:       []byte(CERTIFICATE),
					v1.TLSPrivateKeyKey: []byte(RSA_PRIVATE_KEY),
				},
			}),
			obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
			},
			want: true,
		},
		"remove service": {
			builder: builder(&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "default",
				},
			}),
			obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "default",
				},
			},
			want: true,
		},
		"remove ingress": {
			builder: builder(&v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress",
					Namespace: "default",
				},
			}),
			obj: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress",
					Namespace: "default",
				},
			},
			want: true,
		},
		"remove ingress incorrect ingressclass": {
			builder: builder(&v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			}),
			obj: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			},
			want: false,
		},
		"remove ingressroute": {
			builder: builder(&ingressroutev1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute",
					Namespace: "default",
				},
			}),
			obj: &ingressroutev1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute",
					Namespace: "default",
				},
			},
			want: true,
		},
		"remove ingressroute incorrect ingressclass": {
			builder: builder(&ingressroutev1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			}),
			obj: &ingressroutev1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			},
			want: false,
		},
		"remove httpproxy": {
			builder: builder(&projcontour.HTTPProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute",
					Namespace: "default",
				},
			}),
			obj: &projcontour.HTTPProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute",
					Namespace: "default",
				},
			},
			want: true,
		},
		"remove httpproxy incorrect ingressclass": {
			builder: builder(&projcontour.HTTPProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			}),
			obj: &projcontour.HTTPProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingressroute",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			},
			want: false,
		},
		"remove unknown": {
			builder: builder("not an object"),
			obj:   "not an object",
			want:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.builder.CacheRemove(tc.obj)
			if tc.want != got {
				t.Fatalf("Remove(%v): expected %v, got %v", tc.obj, tc.want, got)
			}
		})
	}
}

func testLogger(t *testing.T) logrus.FieldLogger {
	log := logrus.New()
	log.Out = &testWriter{t}
	return log
}

type testWriter struct {
	*testing.T
}

func (t *testWriter) Write(buf []byte) (int, error) {
	t.Logf("%s", buf)
	return len(buf), nil
}
