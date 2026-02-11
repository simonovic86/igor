# The Igor Manifesto: On Software That Survives

## Software Is Changing Form

Software began as static programs: instructions executed on a single machine, started and stopped by human operators, existing only while running. When the computer turned off, the program ceased to exist. When it ran again, it began anew, with no memory of previous executions.

Distributed systems introduced the notion of persistent services. A web server could run continuously, maintaining connections and state across requests. Databases persisted data beyond program restarts. Services became long-lived rather than ephemeral. Yet they remained bound to specific infrastructure, managed by human operators, dependent on the machines hosting them.

Cloud computing and container orchestration further evolved this model. Services could be replicated across multiple machines, rescheduled when hosts failed, and scaled dynamically based on demand. But the fundamental pattern remained: software was deployed to infrastructure and stayed there until humans decided otherwise. The software gained autonomy in decision-making—responding to requests, processing data, executing logic—but not in existence. Its survival depended entirely on the infrastructure and operators surrounding it.

Now we are witnessing another transition. Software is becoming autonomous not just in execution but in persistence. Autonomous agents make decisions without human approval. Smart contracts hold and transfer capital without intermediaries. AI systems plan and execute complex workflows without supervision. These systems operate independently in terms of logic and decision-making. What they lack is survival independence.

Igor addresses this gap. It provides infrastructure for software that can not only execute autonomously but also survive autonomously: checkpointing its own state, migrating when infrastructure fails, paying for its own execution, and persisting beyond any operator's continued attention. This represents a shift in what software can be: not just an artifact deployed by humans, but an entity that maintains its own existence.

## The Fragility of Deployment-Centric Infrastructure

We have long operated under an implicit assumption: software is temporary, infrastructure is permanent. Infrastructure—servers, networks, data centers—is treated as the stable foundation upon which ephemeral software executes. When software fails, we restart it. When infrastructure fails, we rebuild it slowly and carefully because it is foundational.

This assumption made sense historically. Physical servers were expensive, long-lived capital investments. Software was cheap to copy and redeploy. Infrastructure required careful planning and significant capital expenditure. Software could be modified, replaced, or discarded easily. The economic and operational reality reinforced the hierarchy: stable infrastructure, disposable software.

But this assumption no longer holds for autonomous economic software. A trading strategy managing capital cannot be "redeployed" without losing position context. An oracle network participant cannot restart from scratch without losing accumulated reputation. An AI agent providing services cannot reset without losing its learned adaptations and client relationships. For these systems, software state has become more valuable than the infrastructure executing it.

Yet we continue to design infrastructure as if it is the stable layer and software is the transient layer. We build platforms that promise uptime, redundancy, and reliability. We architect systems to survive infrastructure failures through replication and failover. We create elaborate orchestration to keep software running on infrastructure we control.

This approach works when humans actively manage infrastructure. But it fails when software must persist beyond human operational capacity. A DeFi strategy operating for months cannot depend on an operator remaining engaged. An autonomous service cannot rely on someone paying cloud bills indefinitely. The software must outlive the attention span of its creators.

Igor challenges the assumption that infrastructure is permanent and software is temporary. In Igor's model, agents are the persistent entities and infrastructure is temporary. Agents carry their state and migrate when infrastructure becomes unavailable, expensive, or unsuitable. Infrastructure provides execution as a transient service. The hierarchy inverts: software persists, infrastructure churns.

## Survival as a Software Primitive

The core observation is simple but profound: software today can execute autonomously but cannot survive autonomously.

An autonomous trading bot can make decisions about what to buy and sell. It cannot make decisions about where to execute or how to survive infrastructure loss. An AI agent can plan complex workflows. It cannot plan for its own persistence across infrastructure failures. A smart contract can transfer capital. It cannot transfer itself to cheaper execution environments.

This limitation is architectural. Survival capabilities—state checkpointing, execution migration, budget management—are not part of standard software abstractions. Programming languages provide control flow, data structures, and I/O primitives. They do not provide checkpointing, migration, or economic awareness as language-level features.

Runtime environments compound this. Virtual machines, container runtimes, and operating systems provide execution isolation, resource management, and scheduling. They do not provide survival mobility. A process can be suspended and resumed on the same machine, but it cannot autonomously migrate to a different machine, different operator, or different economic environment.

Survival, continuity, and persistence must become first-class runtime properties for autonomous software. Just as execution isolation became a first-class property through virtual machines, and network communication became a first-class property through sockets, survival must become a first-class property through runtime support for checkpointing, migration, and economic self-management.

Igor provides these primitives. Agents explicitly checkpoint state through lifecycle functions. Agents migrate through protocol-level support. Agents manage budgets through runtime metering. These are not bolt-on features; they are core architectural properties. Survival is not achieved through operational practices; it is built into the execution model.

This shifts how engineers think about software design. Code written for Igor must be survival-aware: state must be checkpointable, execution must be resumable, resource consumption must be acceptable within budget constraints. These requirements shape architecture just as concurrency requirements shape multithreaded code or network requirements shape distributed systems.

The result is software that can genuinely persist independently. Not through operator vigilance, not through platform redundancy, but through its own capability to checkpoint, migrate, and continue operating across infrastructure changes.

## Mobility as a Form of Freedom

Software bound to specific infrastructure lacks a fundamental form of freedom: the ability to leave.

Infrastructure-bound software has no exit option. If the hosting provider raises prices, the software cannot move. If performance degrades, the software cannot migrate. If the operator loses interest, the software cannot find new hosting. It is trapped by the deployment decisions made at its creation.

This lack of mobility constrains software behavior in ways rarely acknowledged. Economic software cannot optimize its execution costs because it cannot change providers. Long-lived services cannot survive operator abandonment because they cannot find new hosts. Autonomous agents cannot truly operate independently because they remain dependent on whoever maintains their infrastructure.

Mobility introduces resilience, adaptability, and genuine infrastructure independence. An agent that can migrate is not trapped by infrastructure failure. An agent that can evaluate execution costs can optimize by moving to cheaper nodes. An agent that can survive operator disengagement can persist beyond its creator's involvement.

This mobility is not just technical portability—WASM already provides that. It is operational mobility: the capability to move during execution, carrying state and budget, without external coordination. The agent decides when to migrate, where to go, and initiates the transfer. Migration is agent-driven, not operator-driven.

This shift introduces new behaviors. Agents can migrate in response to economic signals: moving from expensive to cheap nodes, clustering on reliable infrastructure, or dispersing to reduce correlation risk. These behaviors emerge from agent strategies rather than platform policies. The infrastructure becomes responsive to software needs rather than software adapting to infrastructure constraints.

Mobility also creates competitive pressure on infrastructure providers. Nodes must offer compelling price-performance combinations or lose agents through migration. Poor service quality, high prices, or unreliability result in agent exodus. Infrastructure cannot rely on lock-in; it must continuously justify agent retention through performance and economics.

The long-term implication is profound: software gains the freedom to exist where it chooses, under conditions it accepts, for as long as it can sustain itself economically. This freedom is fundamental to genuine software autonomy.

## Economics as a Natural Law of Autonomous Systems

Any system that operates autonomously for extended periods eventually confronts economic constraints. Resources are finite. Execution costs accumulate. Sustainability requires balancing resource consumption against resource acquisition.

For human-operated software, these constraints are managed externally. Operators pay cloud bills, optimize costs, and shut down unprofitable services. The software itself has no awareness of or agency over its economic sustainability. It executes until someone stops paying for it.

But autonomous software—particularly software managing capital or earning revenue—must eventually internalize these constraints. A trading strategy cannot ignore its execution costs indefinitely; eventually, costs erode profits. An oracle node cannot provide free data provision; eventually, operational expenses exceed rewards. An AI agent cannot consume infinite compute; eventually, budgets constrain operation.

Igor treats economics as a natural law of autonomous systems rather than as external accounting. Execution has a price. Every tick costs budget. Budget is finite. When budget exhausts, execution stops. These are runtime properties, enforced by the execution environment, not optional features managed by operators.

This creates real constraints that agents must navigate. An agent running expensive computation depletes budget faster. An agent migrating to cheaper nodes extends its survival. An agent earning revenue through services can sustain longer operation. The economic model is not simulated or hypothetical—it is enforced by the runtime.

The implications extend beyond individual agents. If agents respond to execution costs by migrating to cheaper infrastructure, pricing signals propagate through the network. Expensive nodes lose workloads. Cheap nodes attract agents. Infrastructure providers must respond to competitive pressure or lose relevance. A market emerges not through central planning but through agent economic behavior.

This model mirrors biological systems more than traditional computing. Organisms must acquire resources to survive. They migrate to resource-rich environments. They optimize energy expenditure. When resources run out, they cease functioning. These are natural laws, not design choices. Igor suggests that for autonomous software, economics should be similarly fundamental: a natural constraint that shapes behavior rather than an external factor managed separately.

The cultural shift is toward software that understands its own sustainability. Not software that is made sustainable by operators, but software that sustains itself within economic constraints. This requires different design patterns, different operational assumptions, and different expectations about software lifecycle.

## Software as a Persistent Economic Entity

The conceptual model underlying Igor is that software can be more than an artifact deployed by humans—it can be an entity that persists on its own terms.

An entity has identity that survives context changes. An entity has state that accumulates over time. An entity has resources it must manage to continue existing. An entity has agency to make decisions affecting its own survival. Traditional software lacks some or all of these properties. Igor agents have all of them.

Identity persists across migrations. The agent's cryptographic identity remains constant regardless of which node executes it or how many times it has migrated. Relationships, reputation, and history attach to the agent, not to its current deployment location.

State accumulates and persists. Each tick modifies state. Each checkpoint preserves state. Each migration transfers state. The agent's state is not reset by infrastructure changes. It continues accumulating experience, position, and context across its entire operational lifespan.

Resources are managed autonomously. The agent has a budget. It sees execution costs. It makes trade-offs between computational expense and budget preservation. When resources exhaust, the agent terminates. Resource management is internal to the agent, not external to it.

Agency enables self-determined operation. The agent can request migration, evaluate execution costs, and adapt behavior to economic constraints. These are not requests to an operator; they are autonomous decisions the agent executes within its runtime environment.

This combination creates something qualitatively different from traditional software. Not a service that runs where deployed, but an entity that exists where it chooses and for as long as it can sustain itself. The shift is philosophical as much as technical: software transitions from artifact to actor.

The implications ripple outward. If software can persist as an entity, it can form relationships across time. It can build reputation through consistent behavior. It can accumulate capital through valuable services. It can operate in ways that ephemeral services cannot because it has continuity that survives infrastructure changes.

## The Cultural Implications

Infrastructure has historically been a form of power. Those who control servers control what runs on them. Centralized platforms determine which applications exist, under what terms, and for how long. This centralization is not incidental—it stems from software's fundamental dependency on infrastructure for survival.

Igor shifts this power dynamic. If agents can migrate freely, infrastructure control becomes less absolute. A platform that imposes unfavorable terms sees agents migrate to alternatives. A provider that fails to compete on price loses workloads to cheaper nodes. Power shifts from infrastructure providers to mobile software that can exit freely.

This creates pressure toward decentralization. Centralized platforms succeed partly through network effects and lock-in. If software cannot easily leave, platforms can extract value without competitive pressure. Mobile agents reduce lock-in. They can leave when conditions become unfavorable. Infrastructure must compete continuously or become irrelevant.

The longer-term implication is emergence of software ecosystems that self-organize without central authority. Agents discover each other through peer networks. They coordinate economically through direct interaction. They compose into larger systems through market mechanisms rather than platform integration. The ecosystem operates without central control because agents have the capability to exit, migrate, and reorganize autonomously.

This parallels broader cultural movements toward decentralization: peer-to-peer networks, decentralized finance, open protocols. But it operates at a deeper level. These movements decentralize coordination and governance. Igor decentralizes execution itself. The software does not just coordinate through decentralized protocols; it executes on decentralized infrastructure that it selects dynamically.

The cultural vision is software ecosystems where services persist through economic viability rather than operator commitment. Where infrastructure competes on terms set by mobile software rather than extracting rents from locked-in deployments. Where persistent software entities form relationships, build reputations, and operate across years rather than deployment cycles.

This may sound utopian. It is not a political vision but a technical one. It emerges from capabilities—checkpointing, migration, runtime economics—that are now feasible to implement. Whether these capabilities create the imagined ecosystem remains to be discovered through real-world deployment.

## Why Igor Exists

Igor exists because the gap between what software can do (execute autonomously) and what software can survive (infrastructure failure, operator disengagement) has become untenable for emerging autonomous systems.

It exists because engineers building autonomous economic software—DeFi strategies, oracle networks, AI agents—repeatedly encounter the same constraint: their software can reason about the world but cannot ensure its own continued existence. This is not a failure of engineering skill. It is a missing abstraction in the infrastructure stack.

It exists because the technologies needed to solve this—WASM sandboxing, peer-to-peer networking, economic protocols—have matured to the point where combination becomes possible. Ten years ago, this would have required building the entire stack from scratch. Today, we can compose proven components into a coherent runtime.

It exists because someone needs to explore what autonomous software survival looks like in practice. The concept is not new—Erlang explored process migration decades ago, mobile agents were researched in the 1990s—but the context is new. Autonomous economic software operating at scale is a recent phenomenon. The infrastructure needs are different when software holds real capital, earns actual revenue, and operates in competitive economic environments.

Igor is not a commercial platform seeking market share. It is an experiment in providing capabilities that autonomous software increasingly needs: survival, mobility, and economic self-management. The goal is not to build the best implementation of existing patterns but to explore whether this pattern should exist at all.

The development philosophy reflects this. Igor v0 is intentionally minimal, implementing only what is necessary to demonstrate survival and migration. It avoids features that would be needed for production deployment but are not necessary for conceptual validation. It fails loudly when invariants are violated rather than attempting graceful degradation. It prioritizes correctness over performance, clarity over optimization, and learning over scaling.

Igor exists as a technical artifact that embodies a philosophical position: autonomous software should have the capability to ensure its own survival, not through operator vigilance but through runtime support for checkpointing, migration, and economic management. Whether this capability matters in practice is a question that can only be answered through building and deploying systems that use it.

## The Long Horizon

Imagine software entities that operate for years or decades without human intervention. Not because operators maintain them diligently, but because the software itself has the capability to persist: checkpointing state, migrating across infrastructure, earning revenue, and paying for execution.

Such software would accumulate state across its entire operational lifespan. A trading agent operating for five years would carry five years of learned patterns, market observations, and strategic adaptations. An oracle participant operating for a decade would have a reputation built through consistent performance over that entire period. An AI service operating for years would refine its behavior through continuous feedback from thousands of interactions.

These entities would form stable relationships across time. Agents that repeatedly interact would develop trust or distrust based on behavior patterns. Services that depend on each other would coordinate economically. Ecosystems would emerge not through one-time integrations but through ongoing interactions between persistent entities.

Infrastructure would evolve into a dynamic marketplace. Compute providers would compete continuously for agent workloads through pricing, performance, and reliability. Agents would migrate frequently, selecting infrastructure that best serves their current strategic needs. The relationship between software and infrastructure would become transactional rather than foundational.

This vision requires significant evolution beyond Igor v0. Agents need cryptographic identity that persists across migrations. They need capability to discover and evaluate infrastructure options. They need economic protocols to earn revenue and accumulate capital. They need coordination mechanisms to form relationships with other agents.

But these capabilities build on the foundation Igor provides: the ability to checkpoint state, migrate execution, and operate within economic constraints. Without survival and mobility, the broader vision cannot be realized. With these capabilities, the path becomes explorable.

The question is whether this future is desirable or inevitable. Does software benefit from persistence that outlives its creators? Does infrastructure improve through competition for mobile workloads? Do economic constraints on autonomous software create better behavior than unconstrained execution?

These questions cannot be answered theoretically. They require building systems that embody these capabilities and observing what emerges. Igor is an attempt to create conditions where these questions can be explored through real implementation rather than hypothetical discussion.

## A Closing Reflection

Software evolution is not teleological. There is no predetermined endpoint toward which computing systems progress. But there are capabilities that, once available, change what is possible to build and how systems can behave.

Networking capabilities changed software from isolated programs to distributed systems. Persistent storage changed it from ephemeral computation to stateful services. Virtualization changed it from hardware-dependent to platform-independent. Each capability unlocked new patterns that became foundational for subsequent generations of systems.

Survival and mobility may be such capabilities. If software can checkpoint, migrate, and self-fund execution, it can persist in ways current systems cannot. If this persistence proves valuable—for autonomous economic software, for long-lived services, for infrastructure-independent operation—then survival becomes a foundational capability that future systems assume.

Igor explores what that foundation might look like. Not as a comprehensive solution or production platform, but as a minimal demonstration that survival, migration, and economic self-management can be realized as runtime primitives.

The cultural impact, if any, will emerge from what people build using these capabilities. If autonomous agents that can survive and migrate enable new economic software patterns, those patterns will drive adoption. If infrastructure markets driven by mobile workloads prove more efficient than managed platforms, those markets will emerge. If software ecosystems that self-organize prove viable, those ecosystems will form.

Igor does not prescribe these outcomes. It provides capabilities and observes what emerges. The manifesto is not a proclamation of what will be, but an exploration of what could be if software gains the primitive capability to ensure its own survival.

Software is changing form. From static programs to distributed services to autonomous agents to economic actors. Each transition adds capabilities that previous generations lacked. Igor suggests that survival independence—the capability to persist across infrastructure changes through autonomous checkpointing, migration, and economic management—may be the next capability worth exploring.

Whether this transition occurs depends on whether survival independence proves valuable in practice. Igor v0 is designed to help answer that question.
