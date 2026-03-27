# Igor Vision: Software That Survives

## The Core Problem

Software today can execute autonomously but cannot survive autonomously.

Autonomous trading strategies manage capital and execute decisions without human intervention, yet they stop when their host server fails. Oracle network participants must maintain continuous uptime, yet they depend entirely on operators to manage infrastructure. AI agents process requests autonomously, yet they cannot survive the failure or abandonment of the infrastructure hosting them.

This dependency represents a conceptual mismatch: software that operates autonomously but deploys statically. The software can reason and act, but it cannot persist independently of the infrastructure chosen at deployment time.

## The Paradigm Shift

Traditional infrastructure operates on a deployment model: developers place software onto infrastructure, and orchestration systems keep it running in that location. Kubernetes reschedules crashed pods. Load balancers route around failed servers. But the software remains tied to the infrastructure chosen for it.

Igor inverts this relationship. Instead of deploying software to infrastructure, Igor enables software to find infrastructure. Agents decide where to execute based on price and availability. Nodes provide execution as a commodity service, not as a permanent home. Infrastructure becomes fungible; agents become persistent.

This shift changes fundamental design assumptions. Software designed for deployment optimizes for fast cold starts and horizontal scaling. Software designed for survival optimizes for state checkpointing and migration speed. The former assumes infrastructure persistence; the latter assumes infrastructure churn.

Migration becomes a first-class operation rather than a disaster recovery mechanism. In Igor, agents migrate when prices change, when budget constrains them, or when infrastructure becomes unavailable. Migration is not exceptional—it is operational.

## Survival as a Runtime Primitive

Survival capabilities—state checkpointing, execution migration, budget management—must become first-class runtime properties, not operational practices layered externally.

Programming languages provide control flow, data structures, and I/O primitives. They do not provide checkpointing, migration, or economic awareness as language-level features. Runtime environments provide execution isolation and scheduling. They do not provide survival mobility.

Igor provides these primitives. Agents explicitly checkpoint state through lifecycle functions. Agents migrate through protocol-level support. Agents manage budgets through runtime metering. Survival is built into the execution model, not bolted on afterward.

This requires different engineering discipline. Code written for Igor must be survival-aware: state must be checkpointable, execution must be resumable, resource consumption must fit within budget constraints. These requirements shape architecture just as concurrency requirements shape multithreaded code.

## Runtime Economics

Most distributed systems treat computation cost as external accounting. The software has no awareness of its execution cost, cannot adjust behavior to reduce costs, and cannot migrate to cheaper infrastructure.

Igor makes economics a runtime primitive. Every tick execution is metered. Cost is calculated as `execution_time × node_price` and deducted from the agent's budget immediately. When budget exhausts, execution stops. The agent knows its burn rate and can reason about economic sustainability.

This visibility enables optimization. Agents running out of budget can reduce computational expense. Agents seeing expensive execution can migrate to cheaper nodes. Economic constraints operate at the runtime level, not the platform level.

If agents pay for execution and nodes charge for resources, a market emerges naturally. Nodes compete on price and reliability. Agents select nodes based on strategic needs. No central platform sets prices. The market discovers equilibrium through agent migration patterns.

## Software as Persistent Entity

Igor enables software that is not just code and state, but a persistent entity with identity, capital, and mobility.

An Igor agent consists of: identity (cryptographic), state (explicit), capital (budget), strategy (WASM code), and mobility (migration capability). Together, these elements form an entity that can persist indefinitely, independent of any operator's continued involvement.

This differs from traditional abstractions. Microservices have state and code but no capital or mobility. Serverless functions have code but no persistent state or identity. Batch jobs lack persistent identity and capital. AI workflows depend on infrastructure provided by operators.

An Igor agent combines these capabilities: persistent identity, portable execution, strategic duration, and capital to fund its own operation. It can be launched with initial budget and strategy, then operate for extended periods without human intervention.

## Cultural Implications

Infrastructure has historically been a form of power. Those who control servers control what runs on them. Centralized platforms determine which applications exist and for how long.

Igor shifts this dynamic. If agents can migrate freely, infrastructure control becomes less absolute. A platform imposing unfavorable terms sees agents migrate to alternatives. A provider failing to compete on price loses workloads. Power shifts from infrastructure providers to mobile software.

This creates pressure toward decentralization. Centralized platforms succeed partly through lock-in. Mobile agents reduce lock-in—they can leave when conditions become unfavorable. Infrastructure must compete continuously.

The longer-term implication is emergence of software ecosystems that self-organize without central authority. Agents discover each other through peer networks. They coordinate economically through direct interaction. Services compose through market mechanisms rather than platform integration.

## Why Now

Several technologies mature simultaneously to make this feasible:

**WASM** has evolved from browser technology to production-ready portable execution format. Runtimes are stable and performant. Modules are platform-independent and deterministic.

**libp2p** has been proven at scale through IPFS and Filecoin. It handles millions of nodes and provides production-grade peer-to-peer networking.

**DeFi** has demonstrated that software can autonomously manage billions in capital. Programmable money enables agents to pay for services and earn revenue.

**AI agents** increasingly operate autonomously, creating demand for infrastructure that enables long-lived operation and economic sustainability.

**Infrastructure commoditization** has made compute a competitive market. Numerous providers offer similar capabilities at similar prices, creating economic incentives for agents to shop for better execution.

These conditions converge now. Missing any one would make autonomous mobile agents substantially harder.

## Long-Term Implications

If Igor's model succeeds, infrastructure could transform from managed platforms to dynamic marketplaces where software selects execution based on economic factors.

Agents would operate for months or years without intervention, earning revenue, paying for execution, and migrating to optimize costs. Infrastructure providers would compete for workloads through pricing and reliability. Software ecosystems would self-organize as agents discover each other and coordinate economically.

This reduces operational overhead for persistent software. Instead of maintaining servers and handling failures, operators launch agents with sufficient capital and let them manage their own infrastructure relationships. Economic models drive efficiency: agents wasting resources exhaust budgets faster; efficient agents survive longer.

Whether this future materializes depends on whether survival independence proves valuable in practice. Igor v0 is designed to answer that question through real implementation and testing.

## What Igor Is

Igor is an experiment in providing capabilities that autonomous software increasingly needs: survival, mobility, and economic self-management. It is not a commercial platform seeking market share. It is an exploration of whether autonomous software survival is valuable and how it might be realized technically.

The development philosophy is minimal and iterative. Igor v0 implements only what is necessary to demonstrate survival and migration. It fails loudly when invariants are violated rather than attempting graceful degradation. It prioritizes correctness over performance, clarity over optimization.

## Igor as Protocol

Igor is a protocol, not a platform. Durable execution platforms — Golem Cloud, Temporal, Restate — answer "where does the agent run?" Igor answers a different question: "who IS the agent?" The competitive moat is agent sovereignty: agent-owned identity (DID), cryptographic lineage (signed checkpoint chain), and true portability (the checkpoint file IS the agent). These platforms are potential deployment targets for Igor agents, not competitors. An Igor agent could run on Golem today, Akash tomorrow, and bare metal next week, with the same DID and unbroken lineage across all three.

The adoption model follows verification, not runtime adoption. Trust primitives spread bottom-up — people verify before they run. The checkpoint format spec is the adoptable artifact (like JWT's RFC 7519). The standalone verifier is the on-ramp (like jwt.io — paste a token, see it decoded). The Igor runtime is one implementation of the protocol (like Auth0 is one implementation of JWT-based auth). Publish the spec, ship the verifier, let verification pull developers toward the runtime.

The initial audience is DeFi. This community already thinks in DIDs, signed proofs, and "don't trust the platform." Provable agent uptime and integrity — "agent lineage: verified, 847 checkpoints, no gaps" — solves a problem DeFi has today and currently addresses with trust-me-bro operator reputation. Agent-owned identity and cryptographic lineage are table stakes in this context, not novel features.

---

Igor exists because the gap between what software can do (execute autonomously) and what software can survive (infrastructure failure) has become untenable for emerging autonomous systems. The technologies needed to solve this have matured to the point where exploration becomes feasible.

Whether autonomous software survival matters in practice can only be answered through building systems that embody these capabilities and observing what emerges.
