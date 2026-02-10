# Igor: Infrastructure for Autonomous Economic Software

**Positioning:** Igor is decentralized runtime infrastructure that enables autonomous software agents to survive infrastructure failure, migrate between execution nodes, and self-fund their operation.

---

## The Opportunity

Three converging trends are creating demand for software that can operate persistently without centralized oversight:

First, autonomous software agents are proliferating. AI systems now plan multi-step workflows, invoke external tools, and execute strategies without human approval. These agents increasingly need to run continuously for extended periods—days, weeks, or longer—to maintain competitive positions, fulfill commitments, or accumulate value through repeated interactions.

Second, decentralized finance has demonstrated that software can hold and manage capital autonomously. Automated market makers control billions in assets. Trading strategies execute thousands of transactions daily. Oracle networks earn fees for data provision. These systems operate economically: they generate revenue, consume resources, and require continuous uptime to remain viable.

Third, infrastructure is becoming commoditized. Compute resources are available from numerous providers at competitive prices. The technical barrier to offering execution services is low. What's missing is software capable of taking advantage of this commodity market by dynamically selecting infrastructure based on price and availability.

These trends converge on a common requirement: software that can persist independently of specific infrastructure providers. Current systems have autonomous decision-making but lack autonomous survival. They can reason about markets but not about their own execution costs. They can manage capital but cannot pay for their own hosting. This gap limits how truly autonomous these systems can become.

## The Problem

Software today is deployed to infrastructure and remains bound to that deployment location until human operators intervene. This creates fragility for long-lived autonomous systems.

An automated trading strategy running on a specific server depends on that server's continued operation. If the server fails, the strategy stops. If the hosting provider has an outage, positions may be liquidated. If the operator forgets to renew services, the strategy disappears. The software can make trading decisions autonomously, but it cannot ensure its own survival.

Oracle network participants face similar constraints. They must maintain continuous uptime to avoid penalties, yet they depend on operators to manage their infrastructure. A failed disk, a network partition, or a configuration error can cause downtime and economic loss. The oracle can autonomously submit data, but it cannot autonomously migrate to backup infrastructure.

AI agents providing services face operational overhead challenges. Running a document analysis agent, a code review service, or a data transformation pipeline requires maintaining servers, handling failures, and managing deployments. The operational burden grows with the number of services. Each service needs dedicated infrastructure management.

The fundamental limitation is that software cannot manage its own infrastructure relationship. It cannot see execution costs, evaluate alternative hosting options, or migrate when conditions change. These decisions remain with human operators. For truly autonomous economic software, this represents a critical dependency that limits operational independence.

## The Igor Solution

Igor provides a runtime where software agents can survive infrastructure changes, migrate between nodes, and pay for execution from internal budgets.

**Survival through checkpointing:** Agents explicitly serialize their state at regular intervals. When infrastructure fails or migration occurs, agents resume from the last checkpoint. State persists independently of the hosting node. An agent running on Node A can stop, checkpoint, and resume on Node B without losing progress.

**Migration between nodes:** Igor implements a peer-to-peer migration protocol. Agents package their WASM code, state checkpoint, and budget, then transfer to a target node via direct stream connection. The target node receives the package, resumes the agent, and confirms success. The source node then terminates its local instance. Migration occurs without centralized coordination.

**Self-funded operation:** Agents carry budgets and pay for execution time. Each tick of execution is metered, and its cost—calculated as execution duration multiplied by node price—is deducted from the agent's budget. When budget exhausts, the agent terminates gracefully. This makes execution cost explicit and enables agents to operate within economic constraints.

**Decentralized node network:** Nodes are peer-to-peer participants that provide execution services. Any operator can run a node, set pricing, and accept migrating agents. There is no central registry, coordinator, or platform. Nodes discover each other through libp2p networking and coordinate directly when transferring agents.

These capabilities combine to enable a new operational model: agents that persist beyond any single infrastructure provider, operate within economic budgets, and migrate as needed to continue execution.

## Differentiation

Igor differs fundamentally from existing infrastructure in three ways.

First, it prioritizes survival over deployment. Traditional platforms optimize for fast cold starts, horizontal scaling, and load distribution. Igor optimizes for state checkpointing, migration speed, and budget management. The design assumption is that infrastructure is temporary and agents must be prepared to move at any time. This is not an operational failure mode—it is normal operation.

Second, runtime economics are built into the execution model, not layered on afterward. Metering is not external monitoring; it is integrated into the tick execution loop. Budget exhaustion is not an administrative event; it is a runtime termination condition. Agents see their execution costs in real time and can adjust behavior accordingly.

Third, the agent is the unit of persistence, not the deployment. In traditional systems, deployments persist across infrastructure changes through orchestration. In Igor, agents persist by carrying their own state and migrating when necessary. The deployment is ephemeral; the agent is persistent.

This combination—survival-centric design, runtime economics, and agent persistence—is architecturally distinct from both traditional cloud platforms and emerging agent frameworks.

## Technology Credibility

Igor v0 is implemented and operational, demonstrating core technical feasibility.

The runtime uses wazero for WASM execution, providing deterministic sandboxing with memory limits and capability restrictions. Agents execute in isolated 64MB memory spaces with no filesystem or network access. Execution timeouts prevent runaway computation.

Peer-to-peer networking uses libp2p, the same transport layer powering IPFS and Filecoin. Nodes discover each other, establish encrypted connections, and transfer agents over multiplexed streams. The migration protocol has been implemented and tested with agents successfully moving between nodes while preserving state.

Checkpoint persistence uses an atomic write pattern (temporary file, fsync, rename) to ensure state durability. Checkpoints include both agent state and budget metadata, allowing complete recovery after node failure. Agents have demonstrated survival across restarts and migrations.

Budget metering operates at nanosecond precision using Go's monotonic clock. Every tick execution is timed, its cost calculated, and budget deducted. Agents terminate automatically when budget exhausts, with final checkpoints preserving terminal state.

The implementation is minimal by design—approximately 3,000 lines of Go code and 200 lines of agent code. This validates that the core concepts can be realized without architectural complexity.

## Market Potential

Igor could enable several infrastructure markets that do not currently exist.

**DeFi automation infrastructure:** Decentralized finance strategies could operate as persistent agents rather than operator-managed bots. A yield optimization strategy, liquidation bot, or arbitrage engine could run autonomously across nodes, migrating to cheaper execution when margins compress. The agent earns trading fees and pays for compute from those fees, self-sustaining without operator involvement.

**AI agent runtime ecosystems:** As AI agents become more sophisticated and economically valuable, they will need execution infrastructure independent of their creators. An agent providing specialized analysis, data transformation, or decision support could operate as a persistent service, accepting requests from multiple clients, earning revenue, and paying for its own hosting.

**Decentralized compute markets:** If agents can migrate based on price, infrastructure providers must compete for workloads. This creates spot markets for compute where pricing adjusts based on supply and demand. Underutilized nodes lower prices to attract agents. Overloaded nodes raise prices to throttle demand. The market operates continuously without centralized price-setting.

**Autonomous service infrastructure:** Services that must operate 24/7 without human intervention—monitoring agents, data pipelines, continuous processing tasks—could run as Igor agents. They would checkpoint regularly, migrate on failure, and continue operation across infrastructure changes. This reduces operational overhead for always-on services.

The economic scale of these markets is difficult to project because they do not currently exist in this form. But the underlying demand is visible: billions spent on DeFi infrastructure, rapid growth in AI agent deployment, and increasing automation of software operations.

## Business Model Potential

Several monetization approaches could build on Igor infrastructure.

**Runtime transaction fees:** Taking a small percentage of migration transactions or checkpoint operations could generate revenue as agent mobility increases. This aligns incentives: more agent activity generates more fee revenue.

**Node marketplace participation:** Operating a marketplace or discovery layer that helps agents find suitable nodes could charge listing fees, transaction fees, or subscription fees. Nodes pay for visibility; agents gain discovery.

**Enterprise private deployments:** Organizations running autonomous agents in trusted environments could deploy private Igor networks. This provides survival and migration capabilities without exposing agents to public infrastructure. Enterprise licensing or support subscriptions become viable.

**Protocol-level participation:** If Igor becomes infrastructure for autonomous agent ecosystems, there may be opportunities to participate at the protocol level through governance tokens, stake-based coordination, or value capture mechanisms. This remains speculative and depends on ecosystem development.

These models are possibilities, not commitments. The primary focus for v0 is validating technical feasibility and demonstrating real-world utility.

## Why Now

Five enabling conditions make Igor viable today when it would not have been possible five years ago.

WASM has matured from experimental browser technology to production-ready portable execution format. Runtimes are stable, performant, and widely deployed. The WASM specification is complete and unlikely to fragment. This provides confidence that agents compiled today will run on infrastructure deployed years from now.

libp2p has been proven at scale through IPFS and Filecoin deployments. It handles millions of nodes, diverse network topologies, and real-world operational challenges. The protocol stack is mature, maintained, and well-understood. This provides confidence in decentralized networking foundations.

Programmable capital through DeFi has demonstrated that software can autonomously manage billions in assets. Smart contracts, automated market makers, and lending protocols operate without centralized control. This validates that economic software agents are feasible and valuable, creating demand for better execution infrastructure.

AI agent capabilities have advanced rapidly. Modern language models can reason about complex tasks, invoke tools, and execute multi-step plans. The bottleneck is no longer intelligence but persistence and economic sustainability. This creates market pull for infrastructure that enables long-lived autonomous operation.

Infrastructure commoditization has made compute a low-margin competitive market. Numerous providers offer similar capabilities at similar prices. This creates economic incentives for agents to shop for better execution and for providers to compete for agent workloads. A dynamic marketplace becomes viable when infrastructure is fungible.

These conditions converge now. Missing any one would make Igor substantially harder. Having all five creates an opportunity window.

## Long-Term Vision

If Igor's model succeeds, infrastructure could transform from managed platforms to dynamic marketplaces where autonomous software selects execution environments based on economic factors.

Agents would operate for months or years without human intervention, earning revenue from services provided, paying for execution from that revenue, and migrating to optimize costs. Infrastructure providers would compete on price, performance, and reliability to attract agent workloads. Agents would form ecosystems where services compose dynamically, discover each other through peer networks, and coordinate economically.

This would reduce operational overhead for always-on software. Instead of maintaining servers, configuring orchestration, and handling failures, operators would launch agents with sufficient capital and let them manage their own infrastructure relationships. The economic model would drive efficiency: agents that waste resources exhaust budgets faster; agents that optimize survive longer.

The ecosystem could become self-organizing. Agents providing valuable services accumulate capital. Agents consuming more resources than they generate terminate naturally. Infrastructure providers offering good service at competitive prices attract more workloads. Poor providers lose agents through migration. Market forces operate at the software level.

This vision is speculative but grounded. The technology exists. The demand exists. The question is whether the economic model is sufficiently compelling to drive adoption. Igor v0 is designed to answer that question through real-world testing.

## Current Maturity & Next Steps

Igor v0 has completed Phase 1 development. All core capabilities—WASM execution, peer-to-peer migration, checkpointing, and budget metering—are implemented and operational. The system can demonstrate autonomous agents surviving infrastructure changes and migrating between nodes.

The immediate focus is validation through extended testing and early adopter feedback. Can agents run stably for days or weeks? Does migration work reliably under diverse network conditions? Do users find value in autonomous survival capabilities? These questions will inform Phase 2 development priorities.

Phase 2 will focus on agent autonomy: enabling agents to discover nodes, evaluate pricing, and initiate migrations programmatically. Phase 3 will add cryptographic payment receipts and economic protocol enhancements. Phase 4 will focus on production hardening and security.

The development philosophy remains minimal and iterative. Features are added only when validated as necessary. The goal is not to build a comprehensive platform but to prove that autonomous mobile agents are viable and valuable.

---

## Investment Perspective

Igor represents infrastructure for an emerging category: autonomous economic software. If this category grows as DeFi, AI agents, and automation expand, infrastructure enabling software survival and mobility becomes valuable.

The technical risk is low. Core components (WASM, libp2p) are proven technologies. The architecture is minimal and focused. Implementation complexity is manageable.

The market risk is moderate. Demand for autonomous agents is growing, but it is unclear how many will require Igor's specific capabilities. Validation through real-world usage is critical.

The timing risk is low. Enabling technologies are mature now. Waiting would not significantly improve technical foundations but could allow competitors to establish positions.

The opportunity is building infrastructure for software that can persist independently, operate economically, and survive infrastructure churn. If autonomous agents become a significant category, Igor-like capabilities become necessary infrastructure.
