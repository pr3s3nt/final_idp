---
id: PROJECT-GLOSSARY
artifact: glossary
status: current
last_reviewed: 2026-09-17
---

# Glossary

| Term | Meaning |
|---|---|
| Application Definition | Versioned logical definition of an application, its workloads, requirements, configuration definitions, and dependencies. |
| Application Definition Version | Immutable version selected for deployment into an environment. Components retain stable identities across versions. |
| Application Specification | Generated, unresolved application artifact derived from an Application Definition Version, such as `score.yaml`. |
| Environment Configuration | Environment-specific direct values and references to resource/workload outputs. Values are resolved during deployment. |
| Deployment | A requested execution against one application, environment, target, Application Definition Version, and Catalog Version. |
| Deployment target | The place where an application is deployed, either managed cloud infrastructure or a pre-existing internal Kubernetes cluster. |
| Catalog Version | Immutable version of platform-managed Resource Definitions selected for a deployment. |
| Resource Definition | Platform-owned recipe describing how a logical or platform resource is managed or linked for supported contexts. |
| Resource Requirement | Logical infrastructure capability requested by an application, such as PostgreSQL or Redis. |
| Platform Requirement | Infrastructure introduced by the platform/target, such as network or Kubernetes cluster. |
| Resource Instance | Durable record of a managed or linked resource for its application, environment, requirement, and target owner. |
| Workload Instance | Durable record of the currently running state of a workload in an environment and target. |
| Deployment Worker | Background process that claims deployment jobs and executes reconciliation, publication, health checks, output collection, and removal. |
| Delivery Repository | Private repository holding desired state for one application across its deployed environments. |
| CD provider | Provider-neutral integration that publishes desired state and reports delivery status; currently Fleet or Argo CD. |
| Verification record | Immutable evidence from a particular test execution. It does not define required behavior. |
| ADR | Architecture Decision Record containing the context, decision, consequences, and status of an important design choice. |
