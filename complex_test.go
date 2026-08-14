package luceneq

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тексты из статей про разработку для тестовых случаев.

const (
	textGoConcurrency = `Go concurrency model uses goroutines and channels.
Goroutines are lightweight threads managed by the Go runtime. They are created
using the go statement. Channels provide a way for goroutines to communicate
and synchronize. The select statement allows a goroutine to wait on multiple
channel operations. Go approach to concurrency emphasizes communication over
shared memory. The sync package provides primitives for synchronization.`

	textKubernetes = `Kubernetes is an open-source container orchestration platform.
It automates the deployment, scaling, and management of containerized applications.
A Kubernetes cluster consists of a control plane and worker nodes. The control plane
manages the cluster state and schedules workloads. Pods are the smallest deployable
units in Kubernetes. Services provide stable networking for pods. Deployments manage
rolling updates and automatic version upgrades of applications.
Kubectl is the command line tool for Kubernetes. Helm is a package manager.`

	textDevOps = `DevOps combines development and operations to shorten the software
delivery lifecycle. Continuous integration automates the build and test process.
Continuous delivery ensures code can be deployed to production at any time.
Configuration management manages system resources. Monitoring and logging provide
visibility into application performance. Incident response handles production issues.`

	textAPI = `RESTful APIs follow architectural constraints including stateless communication,
client-server separation, and uniform interface. HTTP methods map to CRUD operations:
GET retrieves resources, POST creates resources, PUT updates resources, and DELETE removes
resources. JSON is the most common data format for API requests and responses. API versioning
helps manage backward compatibility. Rate limiting protects servers from abuse. Authentication
tokens verify client identity.`

	textMicroservices = `Microservices architecture breaks applications into small,
independent services. Each service owns its data and exposes an API. Service discovery
allows services to find and communicate with each other. API gateways provide a single
entry point for clients. Circuit breakers prevent cascading failures. Distributed tracing
helps debug requests that span multiple services. Event-driven communication enables
loose coupling between services.`

	textMachineLearning = `Machine learning algorithms learn patterns from data.
Supervised learning uses labeled training data to make predictions. Neural networks
inspired by biological neurons process information in layers. Deep learning uses multiple
hidden layers to learn complex representations. Gradient descent optimizes model parameters.
Overfitting occurs when models memorize training data instead of learning general patterns.
Regularization techniques like dropout and L2 regularization prevent overfitting.`

	textSecurity = `Cybersecurity protects systems and data from unauthorized access.
Encryption transforms data to make it unreadable without the correct key. Two-factor
authentication adds an extra layer of security beyond passwords. Firewalls filter network
traffic based on security rules. Penetration testing simulates attacks to find vulnerabilities.
Security patches fix known vulnerabilities. Zero-trust architecture assumes no implicit trust.`
)

func TestComplexQueries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		query     string
		wantMatch bool
		text      string
	}{
		// ============================================
		// Сложные вложенные запросы
		// ============================================

		{
			name:      "deeply nested: three levels",
			query:     "((goroutines AND channels) OR (channels AND sync)) AND concurrency",
			wantMatch: true,
			text:      textGoConcurrency,
		},
		{
			name:      "deeply nested: match second branch",
			query:     "((goroutines AND pods) OR (channels AND sync)) AND concurrency",
			wantMatch: true,
			text:      textGoConcurrency,
		},
		{
			name:      "deeply nested: all fail",
			query:     "((goroutines AND pods) OR (goroutines AND services)) AND cluster",
			wantMatch: false,
			text:      textKubernetes,
		},

		// ============================================
		// Смешанные операторы с приоритетами
		// ============================================

		{
			name:      "AND OR mixed: AND has higher priority",
			query:     "goroutines OR channels AND sync",
			wantMatch: true,
			text:      textGoConcurrency,
		},
		{
			name:      "AND OR mixed with groups",
			query:     "(goroutines OR channels) AND sync",
			wantMatch: true,
			text:      textGoConcurrency,
		},
		{
			name:      "AND OR mixed: only one side",
			query:     "(goroutines OR channels) AND cluster",
			wantMatch: false,
			text:      textGoConcurrency,
		},

		// ============================================
		// Запросы с NOT и сложной логикой
		// ============================================

		{
			name:      "NOT exclude match",
			query:     "deployment AND NOT azure",
			wantMatch: true,
			text:      textKubernetes,
		},
		{
			name:      "NOT exclude no match",
			query:     "deployment AND NOT azure",
			wantMatch: false,
			text:      "deployment azure kubernetes",
		},
		{
			name:      "NOT with groups match",
			query:     "(integration OR delivery) AND NOT azure",
			wantMatch: true,
			text:      textDevOps,
		},
		{
			name:      "NOT with groups: fail",
			query:     "(integration OR delivery) AND NOT azure",
			wantMatch: false,
			text:      "integration delivery azure monitoring",
		},

		// ============================================
		// Wildcards в сложных запросах
		// ============================================

		{
			name:      "wildcard AND compound",
			query:     "learn* AND (supervised OR deep)",
			wantMatch: true,
			text:      textMachineLearning,
		},
		{
			name:      "wildcard in OR branch",
			query:     "network* OR firewall",
			wantMatch: true,
			text:      textSecurity,
		},
		{
			name:      "wildcard multiple",
			query:     "build* AND test*",
			wantMatch: true,
			text:      textDevOps,
		},
		{
			name:      "wildcard no match",
			query:     "kubernetes* AND azure",
			wantMatch: false,
			text:      textKubernetes,
		},
		{
			name:      "wildcard prefix match",
			query:     "monitor* AND (log* OR observ*)",
			wantMatch: true,
			text:      textDevOps,
		},

		// ============================================
		// Required/Prohibited в сложных запросах
		// ============================================

		{
			name:      "required AND prohibited: match",
			query:     "+container +kubernetes -azure",
			wantMatch: true,
			text:      textKubernetes,
		},
		{
			name:      "required AND prohibited: fail prohibited",
			query:     "+container +kubernetes -azure",
			wantMatch: false,
			text:      "container kubernetes azure deployment",
		},
		{
			name:      "required with OR",
			query:     "+api (REST OR GraphQL)",
			wantMatch: true,
			text:      textAPI,
		},
		{
			name:      "required multiple",
			query:     "+container +kubectl +helm",
			wantMatch: true,
			text:      textKubernetes,
		},

		// ============================================
		// Четыре уровня вложенности
		// ============================================

		{
			name:      "four levels nested: match first",
			query:     "((services AND discovery) AND (API AND gateway)) AND microservices",
			wantMatch: true,
			text:      textMicroservices,
		},
		{
			name:      "four levels nested: match second",
			query:     "(((services AND discovery) OR (tracing AND debug)) AND gateway) AND microservices",
			wantMatch: true,
			text:      textMicroservices,
		},
		{
			name:      "four levels nested: fail",
			query:     "(((services AND pods) OR (tracing AND missing)) AND gateway) AND microservices",
			wantMatch: false,
			text:      textMicroservices,
		},

		// ============================================
		// Фразы в сложных запросах
		// ============================================

		{
			name:      "phrase in AND compound",
			query:     `"container orchestration" AND kubernetes`,
			wantMatch: true,
			text:      textKubernetes,
		},
		{
			name:      "phrase in OR compound",
			query:     `"stateless communication" OR "client-server"`,
			wantMatch: true,
			text:      textAPI,
		},
		{
			name:      "phrase with AND NOT",
			query:     `"rolling updates" AND NOT azure`,
			wantMatch: true,
			text:      textKubernetes,
		},
		{
			name:      "phrase with AND NOT: fail",
			query:     `"rolling updates" AND NOT azure`,
			wantMatch: false,
			text:      `"rolling updates" azure version`,
		},

		// ============================================
		// Длинные цепочки AND
		// ============================================

		{
			name:      "long AND chain: match",
			query:     "microservices AND services AND api AND gateway",
			wantMatch: true,
			text:      textMicroservices,
		},
		{
			name:      "long AND chain: fail one",
			query:     "microservices AND services AND pods AND gateway",
			wantMatch: false,
			text:      textMicroservices,
		},

		// ============================================
		// Длинные цепочки OR
		// ============================================

		{
			name:      "long OR chain: match one",
			query:     "docker OR containerd OR container",
			wantMatch: true,
			text:      textKubernetes,
		},
		{
			name:      "long OR chain: match last",
			query:     "helm OR kubectl OR container",
			wantMatch: true,
			text:      textKubernetes,
		},
		{
			name:      "long OR chain: fail all",
			query:     "docker OR kubectl OR azure",
			wantMatch: false,
			text:      "helm cluster monitoring",
		},

		// ============================================
		// Смешанные длинные запросы
		// ============================================

		{
			name:      "mixed: long AND with group OR",
			query:     "security AND (encryption OR firewall) AND (two factor OR password)",
			wantMatch: true,
			text:      textSecurity,
		},
		{
			name:      "mixed: long AND with group OR: fail",
			query:     "security AND (encryption OR firewall) AND (password OR token)",
			wantMatch: false,
			text:      "security encryption patch network",
		},
		{
			name:      "mixed: group AND NOT",
			query:     "(learning OR training) AND NOT (supervised OR deep)",
			wantMatch: true,
			text:      "machine learning dropout gradient",
		},

		// ============================================
		// Запросы с wildcard и фразой
		// ============================================

		{
			name:      "wildcard AND phrase",
			query:     "neural* AND \"hidden layers\"",
			wantMatch: true,
			text:      textMachineLearning,
		},
		{
			name:      "wildcard OR phrase",
			query:     "deploy* OR \"rolling updates\"",
			wantMatch: true,
			text:      textKubernetes,
		},

		// ============================================
		// Очень сложные: 5+ операторов
		// ============================================

		{
			name:      "very complex: 6 terms AND/OR",
			query:     "(goroutines OR channels) AND (sync OR runtime) AND (select OR go)",
			wantMatch: true,
			text:      textGoConcurrency,
		},
		{
			name:      "very complex: with NOT match",
			query:     "(container OR pods) AND (kubernetes AND cluster) AND NOT azure",
			wantMatch: true,
			text:      textKubernetes,
		},
		{
			name:      "very complex: with NOT fail",
			query:     "(container OR pods) AND (kubernetes AND cluster) AND NOT azure",
			wantMatch: false,
			text:      "container pods kubernetes cluster azure",
		},

		// ============================================
		// Cross-domain: разные тексты
		// ============================================

		{
			name:      "cross-domain: API AND security",
			query:     "api AND (authentication OR token)",
			wantMatch: true,
			text:      textAPI,
		},
		{
			name:      "cross-domain fail: wrong domain",
			query:     "api AND (authentication OR token)",
			wantMatch: false,
			text:      textKubernetes,
		},

		// ============================================
		// Edge cases с группами
		// ============================================

		{
			name:      "group OR single AND",
			query:     "(goroutines OR channels) AND runtime",
			wantMatch: true,
			text:      textGoConcurrency,
		},
		{
			name:      "group OR single AND: fail",
			query:     "(goroutines OR channels) AND kubernetes",
			wantMatch: false,
			text:      textGoConcurrency,
		},
		{
			name:      "AND group OR",
			query:     "runtime AND (goroutines OR channels)",
			wantMatch: true,
			text:      textGoConcurrency,
		},

		// ============================================
		// Множественные NOT
		// ============================================

		{
			name:      "multiple NOT match",
			query:     "security AND NOT (firewall OR encryption) AND (two factor OR zero trust)",
			wantMatch: true,
			text:      "security two factor zero trust monitoring",
		},
		{
			name:      "multiple NOT fail",
			query:     "security AND NOT (firewall OR encryption) AND (two factor OR zero trust)",
			wantMatch: false,
			text:      "security firewall encryption two factor",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			p := NewParser()

			// act
			matcher, err := p.ParseQuery(tt.query)
			require.NoError(t, err, "query: %q", tt.query)

			// assert
			got := matcher.Match(tt.text)
			assert.Equal(t, tt.wantMatch, got, "query: %q, text length: %d", tt.query, len(tt.text))
		})
	}
}

func TestComplexQueryParsing(t *testing.T) {
	t.Parallel()

	t.Run("parses without error", func(t *testing.T) {
		t.Parallel()

		p := NewParser()
		complexQueries := []string{
			"(((a AND b) OR c) AND d) AND e",
			"(x OR y) AND (z OR w) AND v",
			"+required (term1 OR term2) AND -excluded",
			"(phrase1 OR \"phrase two\") AND term3",
			"(learn* OR train*) AND (data OR model)",
			"goroutines AND (channels OR sync) AND NOT runtime",
			"+containers +kubernetes -azure",
		}

		for _, query := range complexQueries {
			_, err := p.ParseQuery(query)
			assert.NoError(t, err, "query should parse: %q", query)
		}
	})

	t.Run("invalid queries return errors", func(t *testing.T) {
		t.Parallel()

		p := NewParser()
		invalidQueries := []string{
			"(hello world",    // unclosed paren
			"(hello OR world", // unclosed paren
			"hello world)",    // stray closing paren
		}

		for _, query := range invalidQueries {
			_, err := p.ParseQuery(query)
			assert.Error(t, err, "query should fail: %q", query)
		}
	})
}
