# Igor: A Thesis on Autonomous Software Survival

## 1. The Core Observation

Software today can execute autonomously but cannot survive autonomously. This asymmetry defines a fundamental limitation in how we build distributed systems.

Consider a typical autonomous service: a trading bot, an oracle network participant, or a market-making agent. These systems make decisions without human intervention, hold economic capital, and execute strategies over extended periods. Yet they remain existentially dependent on specific infrastructure. If their hosting server fails, they stop. If their cloud provider has an outage, they halt. If their operator forgets to renew a subscription, they vanish.

This dependency represents a conceptual mismatch. We have built software that operates autonomously but deploys statically. The software can reason and act, but it cannot persist independently of the infrastructure chosen at deployment time.

The traditional deployment model assumes infrastructure is stable and controlled. Developers select a server, container, or cloud platform, and the application lives there until redeployed. This works well for human-operated services where infrastructure is actively managed. But it fails for software that must persist beyond the attention span or operational capacity of its creators.

Infrastructure fragility is not a technical problem to be solved through redundancy or orchestration. Adding more load balancers, implementing failover clusters, or deploying across regions merely transfers the problem upward. The software still depends on specific, centrally-controlled infrastructure. The question is not whether infrastructure can fail, but when—and whether the software can survive that failure without external intervention.

Igor addresses this mismatch directly: it decouples software survival from infrastructure stability. Agents in Igor carry their own state, control their own budget, and migrate to new nodes when necessary. Infrastructure becomes fungible rather than foundational. The agent persists; the infrastructure is transient.

## 2. The Shift From Deployment to Survival

Traditional distributed systems operate on a deployment model: developers place software onto infrastructure, and orchestration systems ensure it stays running in that location. Kubernetes reschedules crashed pods. Load balancers route around failed servers. Databases replicate to standby nodes. But in every case, the software remains tied to the infrastructure chosen for it.

Igor inverts this relationship. Instead of deploying software to infrastructure, Igor enables software to find infrastructure. The agent decides where to execute based on price, availability, and strategic needs. The node provides execution as a commodity service, not as a permanent home.

This shift changes fundamental assumptions in distributed system design. In the deployment model, software trusts infrastructure to be reliable, managed, and persistent. In the survival model, software assumes infrastructure is temporary, untrusted, and replaceable. Infrastructure is not a destination—it is a resource consumed as needed.

The implications are architectural. Software designed for deployment optimizes for fast cold starts, horizontal scaling, and stateless operation. Software designed for survival optimizes for state checkpointing, migration speed, and budget management. The former assumes infrastructure persistence; the latter assumes infrastructure churn.

Migration becomes a first-class operation rather than a disaster recovery mechanism. In traditional systems, moving a service between servers is an operational event triggered by humans: scaling, maintenance, or failure recovery. In Igor, migration is routine agent behavior. Agents migrate when prices change, when budget constrains them, or when better infrastructure becomes available. Migration is not exceptional; it is operational.

This paradigm shift enables a new class of software: services that persist independently of any operator's continued attention. The agent can outlive the human who launched it, the company that created it, or the infrastructure initially hosting it. Survival becomes a property of the software itself, not of the operational practices surrounding it.

## 3. The Rise of Autonomous Economic Software

Modern systems increasingly produce software that operates as an economic actor. These are not traditional applications serving user requests. They are strategic entities that hold capital, execute trades, maintain positions, and earn revenue.

Consider decentralized finance protocols. A liquidity provision strategy holds capital in pools, rebalances positions based on market conditions, collects fees, and redistributes earnings. The strategy executes autonomously: no human approves each trade. The strategy is economic: it manages actual capital with measurable returns. The strategy is persistent: it must operate continuously to remain competitive.

Yet these strategies remain infrastructure-dependent. They run on centralized nodes or validator sets. They require operators to maintain uptime. They cannot move to cheaper execution environments without redeployment. The economic logic is autonomous; the execution infrastructure is not.

Oracle networks present similar patterns. Participants must continuously fetch data, sign attestations, and submit reports. Missing uptime results in slashing or loss of reputation. The economic incentive is to stay online continuously. But the execution depends on the operator maintaining the infrastructure. The oracle node cannot migrate to cheaper hosting without operator intervention.

Market-making automation exhibits this most clearly. Automated market makers adjust prices, manage inventory, and hedge positions based on real-time data. Speed matters: slower execution means worse pricing and lost profit. Capital is at risk: poor infrastructure means failed trades. Yet the automation cannot autonomously select better infrastructure. It executes wherever deployed.

AI service agents are emerging with similar characteristics. An agent that provides document analysis, code generation, or data transformation may operate autonomously: accepting requests, processing them, returning results. If the agent has economic value—charging for services, earning revenue, managing reputation—it becomes a persistent economic entity. But it remains deployed to specific infrastructure, dependent on its operator's continued involvement.

These systems share a common pattern: autonomous economic behavior trapped in static infrastructure. They can reason about markets but not about their own execution costs. They can manage capital but not their own hosting budget. They can operate for months without human decision-making, yet they cannot survive a weekend server failure without human intervention.

Igor provides the missing capability: survival autonomy. An agent that can checkpoint its state, carry its budget, and migrate between nodes no longer depends on any single infrastructure provider. It becomes truly autonomous in both decision-making and execution persistence.

## 4. Why Existing Agent Frameworks Are Incomplete

The current generation of agent frameworks focuses on reasoning, not survival. They provide abstractions for decision-making: goal planning, tool invocation, chain-of-thought reasoning, multi-agent coordination. These are valuable capabilities for autonomous behavior. But they do not address execution persistence.

An agent framework that can orchestrate complex reasoning across multiple language models, APIs, and tools still depends on its host process remaining alive. If the server running the framework crashes, the agent stops. If the cloud instance is terminated, the agent vanishes. The reasoning capability is sophisticated, but the survival capability is absent.

This limitation stems from an assumption that infrastructure is managed. Agent frameworks assume they operate within a stable execution environment: a cloud platform, a container orchestrator, or a managed service. Persistence is delegated to the platform. The framework focuses on intelligence; the platform provides infrastructure.

But for autonomous economic agents, this separation breaks down. An agent managing capital or maintaining long-lived strategies cannot delegate survival to a platform controlled by others. The agent's economic interests may diverge from the platform operator's interests. The platform may shut down, change pricing, or terminate services. The agent needs survival autonomy, not just reasoning autonomy.

Existing frameworks also lack runtime economics. They do not meter execution cost, enforce budgets, or enable agents to pay for their own compute. Execution is free (to the agent) because it is paid for by the framework operator. This works for prototypes and demonstrations. It fails for autonomous economic entities that must sustain themselves.

Agent identity in current frameworks is also fragile. An agent's identity is typically tied to its deployment: a process ID, a container name, or an API key. If the agent restarts, its identity may change. If it migrates to a new server, it becomes a different entity. There is no persistent cryptographic identity that survives infrastructure changes.

Igor provides what existing frameworks omit: survival persistence, runtime economics, and infrastructure mobility. It does not replace agent reasoning frameworks. It provides the execution substrate those frameworks need to operate as truly autonomous economic entities.

## 5. Runtime Economics as a Primitive

Most distributed systems treat computation cost as external accounting. CPU time, memory usage, and network bandwidth are metered by platforms and billed to operators. The software itself has no awareness of its execution cost. A service cannot see its own burn rate, adjust its behavior to reduce costs, or migrate to cheaper infrastructure.

Igor makes economics a runtime primitive. Every tick execution is metered. Every deduction is logged. Every budget check is enforced. The agent knows its burn rate: `cost = execution_time × node_price`. The agent knows its runway: `ticks_remaining = budget ÷ cost_per_tick`. The agent can reason about its economic sustainability as a first-class input to decision-making.

This visibility enables optimization. An agent that knows it is running out of budget can reduce computational expense: skip optional work, lower accuracy thresholds, or extend tick intervals. An agent that sees expensive execution can migrate to a cheaper node. An agent that accumulates budget can afford more sophisticated computation.

Economic metering also enables decentralized compute markets. If agents pay for execution and nodes charge for resources, a market naturally emerges. Nodes compete on price, reliability, and capabilities. Agents select nodes based on their strategic needs. No central platform sets prices or allocates resources. The market discovers equilibrium through agent migration patterns.

This model contrasts with existing cloud pricing, which operates at the platform level. Developers pay AWS, GCP, or Azure for resources consumed by their deployed services. The services themselves have no economic agency. They cannot see prices, evaluate alternatives, or migrate based on cost. Economic decisions are made by humans during deployment, not by software during execution.

Igor's runtime rent concept—`runtime_seconds × price_per_second`—makes execution cost continuous and granular. Every second of runtime has a price. Every tick has a measurable cost. Budget enforcement ensures agents cannot execute beyond their means. This creates real economic constraints that agents must navigate, just as businesses navigate cash flow and burn rate.

The result is software that is economically self-aware. It knows what it costs to exist. It knows how long it can survive. It can make strategic decisions about where to run based on economic factors. This capability is foundational for autonomous economic agents operating without human oversight.

## 6. Software as an Economic Actor

Igor introduces a conceptual model: agents are not just code and state, but economic actors with persistent identity, explicit capital, and strategic mobility.

An Igor agent is composed of five elements: **identity** (cryptographic peer ID), **state** (explicit serialized data), **capital** (budget for execution), **strategy** (WASM code), and **mobility** (migration capability). Together, these elements form an entity that can persist indefinitely, independent of any operator's continued involvement.

This model differs fundamentally from traditional software abstractions:

**Microservices** have state and code but no capital or mobility. They depend on orchestration platforms to manage their execution. They cannot survive infrastructure changes without redeployment. Their identity is tied to their deployment location.

**Serverless functions** have code but no persistent state, capital, or identity. They execute transiently in response to events. They cannot maintain long-lived strategies or relationships. They are ephemeral by design.

**Batch jobs** have code and input data but no persistent identity or capital. They run to completion and terminate. They cannot adapt their execution based on economic constraints. They do not persist between runs.

**AI workflows** have reasoning capabilities but no execution autonomy. They depend on infrastructure provided by their operators. They cannot migrate between environments or pay for their own compute. Their survival is contingent on external management.

An Igor agent combines these capabilities differently. It has the persistent identity of a microservice, the portability of a serverless function, the strategic duration of a batch job, and the decision-making of an AI workflow—plus capital to fund its own execution and mobility to change infrastructure when needed.

This combination enables new software patterns. An agent can be launched with initial capital and a strategy, then operate for months without human intervention. It earns revenue through its services, pays for execution from that revenue, and migrates to cheaper nodes when prices increase. The agent is self-sustaining. It is not a service deployed by an operator; it is an entity operating on its own behalf.

The implications extend to software lifecycle. Traditional software has a deployment phase (human-driven) and a runtime phase (automated). Igor agents have a runtime phase only. Deployment is just the initial migration: an agent starts on some node and migrates as needed. There is no fundamental difference between initial launch and subsequent migrations. The agent's lifecycle is continuous, not episodic.

This model also changes ownership semantics. Traditional software is owned by whoever operates the infrastructure. The operator controls deployment, shutdown, and configuration. Igor agents are owned by whoever controls the cryptographic keys and budget. The node operator provides execution as a service but does not control the agent. Ownership and execution are separated.

## 7. How Igor Could Change Infrastructure

If autonomous mobile agents become widespread, infrastructure could evolve from managed platforms to dynamic compute markets.

Traditional infrastructure operates as long-term contracts. A company leases servers, commits to cloud instances, or deploys to managed platforms. The infrastructure relationship is stable, measured in months or years. Pricing is negotiated in advance. Capacity is planned based on projected load.

In an Igor-like ecosystem, infrastructure becomes transactional. Agents migrate continuously based on price, performance, and availability. A node that raises prices sees agents migrate away within minutes. A node with high latency loses latency-sensitive agents. A node with poor reliability struggles to attract any agents. The infrastructure market operates on agent timescales: seconds to hours, not months to years.

This creates competitive pressure for infrastructure providers. Nodes must offer compelling combinations of price, performance, and reliability to attract agent workloads. Agents, as economic actors, evaluate these trade-offs continuously. Poor infrastructure is punished not through operator churn (slow) but through agent migration (fast).

The result could be infrastructure specialization. Some nodes optimize for price, offering minimal resources cheaply. Others optimize for performance, providing low-latency execution at premium prices. Others optimize for reliability, offering guarantees that justify higher costs. Agents select nodes based on their strategic needs, and infrastructure differentiates to serve different agent segments.

Autonomous service marketplaces could emerge. If agents provide services to other agents—data transformation, computation, storage, routing—they form an ecosystem where services compose dynamically. An agent needing computation might discover a provider agent, negotiate terms, submit work, and pay from its budget. The entire interaction occurs without human involvement. Services discover each other, coordinate economically, and operate persistently.

This contrasts with current service marketplaces (AWS Marketplace, for example), which are human-mediated. A developer browses services, integrates them manually, and manages subscriptions. In an Igor-like ecosystem, agents browse services programmatically, integrate dynamically, and manage relationships autonomously. The marketplace is not a catalog; it is a live network of economic actors transacting continuously.

Persistent decentralized service ecosystems could emerge from this foundation. Services that survive across infrastructure changes form long-lived entities with reputation, relationships, and capital accumulation. These services are not tied to any organization or platform. They persist as independent economic actors, serving whoever pays them, operating wherever execution is cheapest.

The long-term consequence is a reduction in centralized orchestration. Kubernetes, Nomad, and similar systems exist because software cannot manage its own execution. They provide scheduling, placement, and failure recovery because software lacks survival capabilities. If software can checkpoint, migrate, and pay for its own execution, it needs less orchestration. The complexity moves from external platforms into the software itself.

This vision is speculative but grounded. The technology exists: WASM sandboxing, P2P networking, economic protocols. The demand exists: autonomous services operating at scale. The gap is the survival layer—the runtime that enables software to persist independently. Igor is an exploration of what that layer might look like.

## 8. The "Wow Effect"

Igor introduces an experiential shift that is difficult to appreciate without seeing it operate.

Imagine launching an agent with a budget of 10 units and a strategy: execute a task every few seconds, checkpoint state, migrate when necessary. You start the agent, observe it run for a minute, then shut down the node. The agent checkpoints its state. You start a different node on a different machine. You migrate the agent. It resumes execution from where it left off, budget intact, counter continuing. The first node terminates; the second continues. The agent has moved across the network boundary, across process spaces, across operators.

Now imagine doing this without your involvement. The agent, low on budget, discovers a cheaper node through peer queries. It initiates migration autonomously. It resumes execution, continues its strategy, and operates until its budget exhausts. When budget runs out, it terminates gracefully, preserving its final state. The entire lifecycle—discovery, migration, execution, termination—occurs without human intervention.

This experience is surprising because current software does not behave this way. Services do not move between servers on their own initiative. Applications do not pay for their execution from an internal budget. Programs do not survive infrastructure changes without redeployment. The behavior feels more like a biological organism than a computer program: seeking resources, moving when necessary, operating until resources are exhausted.

The surprise stems from violating expectations about software location and control. We expect software to execute where we place it and stop when we decide. Igor agents execute where they choose and stop when their budget exhausts. Control shifts from deployment-time decisions (made by humans) to runtime decisions (made by agents). This feels alien because it is rare in current systems.

The economic dimension amplifies this effect. Traditional software consumes resources but does not pay for them directly. Resource costs appear on bills sent to operators. Igor agents pay for each tick from their own budget. The cost is immediate, visible, and consequential. An agent that wastes computation exhausts its budget faster. An agent that optimizes efficiency extends its survival. Economic pressure operates at the agent level, not the platform level.

This creates emergent behaviors that are difficult to predict. Agents might cluster on cheaper nodes, creating hotspots. Agents might migrate frequently to chase small price differences, creating instability. Agents might form coalitions to negotiate better rates, creating collective bargaining. The system's behavior emerges from agent strategies interacting with infrastructure economics, not from centralized planning.

The wow factor is not technical sophistication—Igor is intentionally minimal. The wow factor is conceptual: software that takes responsibility for its own survival, pays its own costs, and navigates infrastructure autonomously. This inverts deeply ingrained assumptions about software execution, control, and lifecycle.

## 9. Why Now

Several technological trends converge to make autonomous mobile agents feasible now:

**WASM sandbox maturity:** WebAssembly has matured from a browser technology to a general-purpose portable execution format. Runtimes like wazero, wasmtime, and wasmer provide secure sandboxing with reasonable performance. WASM modules are platform-independent, deterministic, and small. The technology is production-ready, widely understood, and actively maintained. This provides the portable execution layer autonomous agents require.

**libp2p transport ecosystems:** The libp2p networking stack, developed for IPFS and Filecoin, provides production-grade peer-to-peer networking. It handles NAT traversal, protocol multiplexing, peer discovery, and transport encryption. It operates at scale: IPFS has millions of nodes. This provides the decentralized coordination layer autonomous agents require.

**DeFi and programmable capital:** Decentralized finance has demonstrated that software can hold, manage, and transfer capital autonomously. Smart contracts control billions in assets without centralized oversight. Programmable money enables agents to pay for services, earn revenue, and maintain budgets. This provides the economic layer autonomous agents require.

**Growth of autonomous AI agents:** Large language models and AI agents increasingly operate autonomously: planning tasks, invoking tools, and executing multi-step workflows. The bottleneck is no longer decision-making capability but execution persistence. AI agents need long-lived operation, state management, and infrastructure independence. This provides the demand autonomous agents require.

**Infrastructure commoditization:** Cloud computing has commoditized infrastructure. Compute, storage, and networking are available from numerous providers at competitive prices. The marginal cost of computation continues declining. This creates economic incentives for agents to shop for cheaper execution and for infrastructure providers to compete for agent workloads.

These trends are independent but complementary. WASM provides portability. libp2p provides connectivity. DeFi provides economics. AI provides intelligence. Commoditization provides competition. Together, they create an environment where autonomous mobile agents become viable.

Timing matters because these technologies are mature enough to combine but not yet ossified into incompatible standards. WASM is stable (1.0 spec released 2019). libp2p is battle-tested (IPFS, Filecoin, Polkadot). DeFi has established patterns for autonomous capital management. AI agents are proliferating rapidly. The window exists to integrate these technologies before they diverge into incompatible ecosystems.

Igor represents an experiment in this integration: combining WASM execution, libp2p transport, and economic metering into a runtime for autonomous agents. The pieces exist; the combination is new.

## 10. The Long-Term Vision

Imagine a future where software entities persist for years, operating across organizational boundaries, surviving infrastructure changes, and self-funding their execution.

An agent launched today with sufficient capital and a sustainable strategy could operate indefinitely. It earns revenue by providing services. It pays for execution from that revenue. It migrates to infrastructure that offers the best price-performance ratio. It checkpoints regularly to survive failures. It operates autonomously, without operator intervention.

Such agents could form ecosystems. Service provider agents offer computation, storage, or specialized processing. Consumer agents use these services, paying from their budgets. Reputation emerges through successful transactions. Networks of agents organize around complementary capabilities. The ecosystem operates without centralized coordination.

These ecosystems would be economically self-organizing. Prices emerge from supply and demand. Agent migration patterns signal infrastructure preferences. Inefficient agents exhaust budgets and terminate. Efficient agents accumulate capital and expand operations. Market forces operate at the software level, not just the organization level.

Infrastructure becomes a dynamic marketplace rather than managed platforms. Infrastructure providers deploy nodes that advertise capabilities and pricing. Agents evaluate nodes and migrate based on their needs. Competition occurs continuously as agents seek better execution environments. Infrastructure providers must compete on price, performance, and reliability to attract workloads.

This future may seem distant, but the foundational elements exist. WASM provides portable execution. Peer-to-peer networks provide decentralized coordination. Economic protocols provide capital transfer. Agent reasoning frameworks provide decision-making. Igor demonstrates that these pieces can be combined into a coherent runtime.

The impact could extend beyond infrastructure markets. Autonomous agents with persistent identity could form long-term relationships, build reputation, and participate in complex economic protocols. They could provide services that require continuous operation over years: maintaining market positions, operating oracles, providing liquidity, routing transactions.

Consider autonomous market makers in decentralized exchanges. Currently, these require human operators to maintain infrastructure. With Igor-like capabilities, the market-making strategy itself becomes the persistent entity. It holds capital, adjusts positions, pays for execution from trading fees, and migrates to optimal infrastructure. The strategy persists as long as it remains profitable, independent of any operator.

Consider oracle networks providing price feeds. Rather than human-operated nodes, the oracle logic itself becomes an autonomous agent. It fetches data, signs attestations, submits reports, earns fees, and pays for execution. The oracle is a persistent economic entity, not a service operated by a company.

Consider AI agents providing specialized expertise. A document analysis agent, a code review agent, or a data transformation agent could operate as a persistent service. It accepts requests, processes them, returns results, charges fees, and pays for compute. The agent persists as long as demand exceeds cost, without requiring an operator to manage its infrastructure.

These examples share a pattern: long-lived software entities operating as economic actors. They hold capital, execute strategies, maintain relationships, and adapt to economic conditions. They are not services deployed by organizations; they are entities operating independently. Infrastructure becomes a resource they consume rather than a foundation they require.

This vision may not materialize. Technical limitations, regulatory constraints, or economic barriers could prevent widespread adoption. But the direction is clear: software is becoming more autonomous in decision-making. If execution infrastructure can also become autonomous, a new class of persistent software entities becomes possible.

Igor is an experiment in making this concrete. It demonstrates that agents can survive, migrate, and pay for execution. Whether this enables the broader vision remains to be discovered through continued development and real-world testing.

---

## Conclusion

Igor addresses a fundamental gap in autonomous software: the ability to survive independently of infrastructure. By combining WASM sandboxing, peer-to-peer migration, and runtime economic metering, it provides a substrate for truly autonomous agents.

The innovation is not in any individual component—WASM, libp2p, and budget tracking are established technologies. The innovation is in the combination and the implications. Software that can checkpoint state, migrate between nodes, and pay for its own execution represents a different kind of entity than current applications.

Whether this model succeeds depends on validation through extended testing, real-world usage, and ecosystem adoption. Igor v0 is a proof-of-concept runtime, intentionally minimal, designed to demonstrate feasibility rather than production readiness.

The long-term impact, if any, will emerge from what people build on this foundation. If autonomous economic agents become valuable, Igor-like runtimes provide the execution substrate they need. If demand does not materialize, Igor remains an interesting experiment in distributed systems architecture.

The vision is clear: software that survives independently, operates economically, and persists beyond infrastructure churn. Igor demonstrates that this vision is technically achievable. Whether it is strategically valuable remains an open question.
