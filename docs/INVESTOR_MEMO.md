# Igor: Investor Positioning Memo

## Positioning Snapshot

Igor is runtime infrastructure for autonomous software agents that can survive infrastructure failure, migrate between execution environments, and self-fund their operation.

**What Igor Is:** A decentralized runtime enabling software agents to checkpoint state, migrate across peer nodes, and pay for execution using internal budgets—creating software that persists independently of any infrastructure provider.

**Why Now:** WASM sandbox maturity, proven peer-to-peer networking, and the rise of autonomous economic software (DeFi automation, AI agents) converge to make mobile, self-funding software feasible for the first time.

**Why It Wins:** Igor inverts the infrastructure relationship—agents select and pay for execution rather than depending on operator-managed deployments, creating the foundation for decentralized compute markets driven by mobile software workloads.

---

## The Problem

Software today can execute autonomously but cannot survive autonomously. This creates a critical limitation for emerging autonomous economic systems.

Consider decentralized finance automation: trading strategies, liquidity managers, and arbitrage bots operate continuously, managing real capital and executing without human approval. Yet they remain existentially tied to specific servers. A failed host means stopped execution. An infrastructure outage means missed opportunities or liquidated positions. The software can reason about markets but cannot ensure its own survival.

Oracle networks face similar constraints. Participants must maintain continuous uptime to avoid penalties, yet they depend entirely on operators to manage infrastructure. A single server failure can cause economic loss through slashing or reputation damage. The software executes autonomously but relies on humans for operational continuity.

AI agents providing services demonstrate this gap most clearly. An agent that analyzes documents, generates code, or transforms data can operate autonomously—accepting requests, processing them, returning results—but it requires an operator to maintain the infrastructure. The moment the operator stops paying attention, the service becomes vulnerable to failures that the agent cannot resolve on its own.

The fundamental problem is that current infrastructure couples execution to specific deployment locations. Software cannot migrate when infrastructure becomes expensive, unreliable, or unavailable. It cannot pay for its own hosting from revenue it generates. It cannot survive the failure of the infrastructure it was originally deployed to. For software that must operate persistently without human intervention, this dependency represents a critical architectural constraint.

## The Igor Solution

Igor enables agents to survive infrastructure changes by combining four capabilities that existing systems lack.

First, agents checkpoint their state explicitly at regular intervals. State is serialized to a portable format that includes execution state, budget remaining, and metadata. When infrastructure fails or migration occurs, agents resume from the last checkpoint without losing progress. Survival does not depend on infrastructure durability—it depends on checkpoint availability.

Second, agents migrate between nodes using a peer-to-peer protocol. An agent running on Node A can package its code, state, and budget, then transfer to Node B over a direct network connection. Node B receives the package, resumes the agent, and confirms success. Node A then terminates its local instance. Migration happens without centralized coordination or platform assistance.

Third, agents carry budgets and pay for execution time. Each operation is metered, its cost calculated as execution duration multiplied by the node's price, and that cost deducted from the agent's budget. When budget exhausts, the agent terminates gracefully with a final checkpoint. This makes execution cost explicit and enforceable at the runtime level.

Fourth, nodes form a decentralized network where any operator can provide execution services. Nodes set their own pricing, accept migrating agents, and compete for workloads. There is no central platform, registry, or coordinator. Agents discover nodes through peer-to-peer networking and select execution environments based on price and requirements.

These capabilities combine to create software that persists independently of any single infrastructure provider. An agent can operate for weeks or months, migrating between nodes as needed, paying for execution from its budget, and surviving infrastructure failures through checkpointing.

## What Makes Igor Fundamentally Different

Igor represents a paradigm shift from deployment-centric to survival-centric infrastructure.

Traditional platforms optimize for deploying software to managed infrastructure. They provide scheduling, load balancing, failure recovery, and scaling—all mechanisms to keep software running in its designated location. The software is passive; the platform is active. Persistence comes from platform reliability, not software capability.

Igor inverts this model. Agents are active; infrastructure is passive. Agents checkpoint their own state, decide when to migrate, and pay for their own execution. Infrastructure provides computation as a commodity service. The agent selects infrastructure; infrastructure does not control the agent. Persistence comes from agent capability, not platform reliability.

This inversion has architectural consequences. Software designed for Igor optimizes differently: fast checkpointing matters more than fast cold starts; migration speed matters more than horizontal scaling; budget efficiency matters more than peak performance. The design assumptions are fundamentally different because the threat model is different. Traditional software fears application bugs. Igor agents fear infrastructure loss.

Runtime economics become a native primitive rather than external accounting. In cloud platforms, execution cost is metered by the platform and billed to operators. The software has no visibility into or control over these costs. In Igor, execution cost is metered by the runtime and charged to the agent immediately. The agent sees costs, evaluates trade-offs, and makes decisions based on economic factors. Economics are not external—they are integrated into the execution model.

This combination—survival-first design and native runtime economics—differentiates Igor architecturally from both traditional cloud infrastructure and emerging agent frameworks. It is not a better implementation of existing patterns; it is a different pattern addressing different constraints.

## Proof of Feasibility

Igor v0 is implemented and operational, demonstrating that the core concepts are technically viable.

The system uses wazero for WASM execution, providing portable sandboxed computation with memory limits and capability restrictions. Agents execute in isolated 64MB environments with no filesystem or network access, preventing malicious behavior while maintaining execution portability across platforms.

Peer-to-peer networking uses libp2p, the same transport stack powering IPFS and Filecoin at scale. Nodes discover peers, establish encrypted connections, and transfer agents over multiplexed streams. The migration protocol has been tested with agents successfully moving between nodes while preserving state and budget.

State persistence uses atomic checkpoint writes with explicit durability guarantees. Checkpoints survive process crashes and system reboots. Agents have demonstrated survival across multiple restarts and migrations without state loss or budget corruption.

Runtime metering operates with nanosecond precision, timing every execution and calculating costs immediately. Budget enforcement terminates agents automatically when funds exhaust. The economic model is not theoretical—it is implemented and enforced by the runtime.

The implementation is minimal: approximately 3,000 lines of Go runtime code and 200 lines of agent code demonstrate core functionality. This validates that the architecture does not require significant complexity to realize the core concepts.

## Market Wedge

Igor's initial market is **DeFi automation infrastructure**, where autonomous economic software already operates at significant scale but lacks survival independence.

Decentralized finance strategies—yield optimizers, liquidation engines, arbitrage bots—currently run on operator-managed servers or validator nodes. These strategies hold capital, execute trades continuously, and earn fees. But they depend on infrastructure maintained by their operators. If infrastructure fails, capital is at risk. If operational costs rise, profitability suffers. The strategies cannot autonomously optimize their execution costs or survive infrastructure changes.

Igor enables these strategies to operate as persistent agents. A yield optimization strategy could run as an Igor agent: earning trading fees, paying for execution from those fees, migrating to cheaper nodes when margins compress, and surviving infrastructure failures through checkpointing. The strategy becomes self-sustaining, requiring minimal operator involvement beyond initial deployment.

This wedge market has several advantages. First, DeFi automation already exists at scale, validating demand for autonomous persistent software. Second, economic incentives are clear: execution cost directly impacts profitability, creating motivation to optimize. Third, infrastructure independence has immediate value: uptime penalties and liquidation risks make survival capability economically meaningful.

Secondary expansion markets include AI agent runtime infrastructure and oracle network automation. AI agents providing services could operate as Igor agents, accepting requests, earning revenue, and self-funding execution. Oracle nodes could run as persistent agents, continuously providing data feeds while managing their own hosting costs. These markets share the pattern of autonomous software requiring persistent operation but lacking survival infrastructure.

## Business Model Potential

Several credible monetization paths exist for Igor infrastructure.

**Runtime transaction fees:** Charging a small percentage on agent migrations or checkpoint operations could generate revenue as agent mobility increases. This aligns with ecosystem growth: more agents, more migrations, more revenue. The model scales with ecosystem adoption without requiring platform lock-in.

**Node marketplace participation:** Operating discovery infrastructure that helps agents find suitable execution nodes could charge listing fees to node operators or convenience fees to agents. This creates a business opportunity adjacent to the core protocol without requiring control of agent execution itself.

**Enterprise private deployments:** Organizations running sensitive autonomous agents may prefer private Igor networks rather than public infrastructure. This creates opportunity for enterprise licensing, support subscriptions, and managed deployment services. The business model resembles traditional enterprise infrastructure sales but applied to autonomous agent hosting.

**Protocol-level value capture** remains speculative but could emerge if Igor becomes foundational infrastructure. Mechanisms might include governance participation, stake-based coordination, or transaction fee models. This path depends on ecosystem maturity and is not a primary near-term focus.

The monetization strategy prioritizes ecosystem growth over immediate revenue extraction. Early focus is on validating technical viability and demonstrating real-world utility. Business model optimization follows once product-market fit is established.

## Defensibility

Igor's defensibility comes from protocol-level positioning and network effects rather than proprietary technology.

As a protocol, Igor becomes more valuable as more agents and nodes adopt it. Agents benefit from more execution options; nodes benefit from more potential workloads. This creates network effects where ecosystem growth increases utility for all participants.

Switching costs emerge through agent standardization. Agents written for Igor's execution model (checkpointing, migration, budgets) cannot easily run on alternative infrastructure without rewriting. As more agents adopt Igor's patterns, alternative runtimes must provide compatible interfaces or lose access to that agent ecosystem.

Runtime accounting primitives create operational moats. Once agents depend on metered execution, budget enforcement, and economic decision-making, those capabilities become necessary infrastructure. Future systems must provide equivalent functionality or remain incompatible with economically self-aware agents.

Future layers—reputation systems for nodes, payment receipt verification, settlement protocols—could create additional defensibility. But these remain hypothetical. Current defensibility stems from protocol standardization and ecosystem network effects.

## Near-Term Roadmap Direction

Three immediate priorities focus on validation and ecosystem readiness:

**Capability validation and policy enforcement:** Enabling nodes to advertise capabilities and reject incompatible agents. This allows infrastructure specialization (compute-intensive nodes, memory-optimized nodes, etc.) and creates a foundation for agent-node matching markets.

**Multi-node mobility optimization:** Reducing migration latency, improving checkpoint efficiency, and enabling agents to evaluate multiple migration targets. This makes dynamic infrastructure selection practical for latency-sensitive applications.

**Payment receipt verification:** Adding cryptographic receipts for execution time and cost. This enables auditable payment trails and builds toward trustless settlement, reducing reliance on node honesty assumptions.

These milestones are sequential and focused. Each builds on validated functionality rather than speculating about distant features. The development philosophy remains minimal and iterative.

## Closing Vision

If Igor's model succeeds, infrastructure could evolve from managed platforms to dynamic marketplaces where software selects execution based on economic factors.

Agents would operate persistently across heterogeneous networks, migrating to optimize costs, earning revenue from services provided, and paying for infrastructure from that revenue. Infrastructure providers would compete for agent workloads through pricing, performance, and reliability. Software ecosystems would self-organize: valuable agents accumulate capital and expand; inefficient agents exhaust budgets and terminate naturally.

This creates infrastructure markets driven by software rather than by human operators. Compute pricing becomes dynamic, adjusting continuously as agents migrate in response to price changes. Infrastructure specialization emerges as providers differentiate on capabilities rather than brand. Operational overhead for persistent software decreases as agents manage their own infrastructure relationships.

The opportunity is building the foundational layer for autonomous economic software. If this category grows as projected—through DeFi expansion, AI agent proliferation, and increasing automation—infrastructure enabling software survival, mobility, and economic self-management becomes necessary. Igor is positioned to provide that infrastructure.

The immediate goal is validation through real-world usage. Can agents operate stably for extended periods? Do users find value in survival and migration capabilities? Do economic incentives create viable compute markets? Igor v0 is designed to answer these questions through practical deployment and testing.

The long-term potential depends on ecosystem adoption. If autonomous economic software becomes widespread, Igor-like capabilities become foundational infrastructure. If adoption remains limited to niche applications, Igor remains an interesting experiment in distributed systems architecture. The next twelve months will provide evidence for which trajectory is more likely.
