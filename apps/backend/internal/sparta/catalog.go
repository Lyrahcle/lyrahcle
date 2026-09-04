package sparta

import "sort"

const (
	CatalogVersion   = "lyrahcle-2026.09"
	FrameworkName    = "SPARTA"
	FrameworkOwner   = "The Aerospace Corporation"
	FrameworkURL     = "https://sparta.aerospace.org/"
	CatalogScopeNote = "Manually maintained curated subset of the published SPARTA matrix covering the techniques Lyrahcle can produce evidence for. Reconcile identifiers with the published matrix before using them in reports or procurement documents."
)

type Tactic struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Order       int    `json:"order"`
	Description string `json:"description"`
}

type Technique struct {
	ID          string `json:"id"`
	TacticID    string `json:"tactic_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Segment     string `json:"segment"`
}

const (
	TacticReconnaissance     = "ST0001"
	TacticResourceDeployment = "ST0002"
	TacticInitialAccess      = "ST0003"
	TacticExecution          = "ST0004"
	TacticPersistence        = "ST0005"
	TacticDefenseEvasion     = "ST0006"
	TacticLateralMovement    = "ST0007"
	TacticExfiltration       = "ST0008"
	TacticImpact             = "ST0009"
)

const (
	SegmentSpace  = "space"
	SegmentGround = "ground"
	SegmentLink   = "link"
	SegmentUser   = "user"
)

var tactics = []Tactic{
	{ID: TacticReconnaissance, Name: "Reconnaissance", Order: 1, Description: "Gathering information about the mission, its spacecraft and its communications before acting on it."},
	{ID: TacticResourceDeployment, Name: "Resource Development", Order: 2, Description: "Building or acquiring the infrastructure and capabilities used later in the operation."},
	{ID: TacticInitialAccess, Name: "Initial Access", Order: 3, Description: "Obtaining a first foothold in the ground segment, the link or the spacecraft."},
	{ID: TacticExecution, Name: "Execution", Order: 4, Description: "Running adversary-controlled behaviour on a mission system."},
	{ID: TacticPersistence, Name: "Persistence", Order: 5, Description: "Retaining access across resets, contacts and credential changes."},
	{ID: TacticDefenseEvasion, Name: "Defense Evasion", Order: 6, Description: "Avoiding detection by operators, fault management and monitoring."},
	{ID: TacticLateralMovement, Name: "Lateral Movement", Order: 7, Description: "Moving between subsystems, spacecraft or ground systems."},
	{ID: TacticExfiltration, Name: "Exfiltration", Order: 8, Description: "Removing mission data from the system."},
	{ID: TacticImpact, Name: "Impact", Order: 9, Description: "Denying, degrading, disrupting, deceiving or destroying mission capability."},
}

var techniques = []Technique{
	{ID: "REC-0003", TacticID: TacticReconnaissance, Name: "Gather Spacecraft Communications Information", Description: "Collecting downlink and uplink characteristics, frequencies and contact schedules.", Segment: SegmentLink},
	{ID: "REC-0005", TacticID: TacticReconnaissance, Name: "Eavesdropping", Description: "Passively receiving mission links to learn their structure and content.", Segment: SegmentLink},
	{ID: "REC-0008", TacticID: TacticReconnaissance, Name: "Gather Supply Chain Information", Description: "Collecting information about mission components, software and their known weaknesses.", Segment: SegmentGround},
	{ID: "REC-0009", TacticID: TacticReconnaissance, Name: "Gather Mission Information", Description: "Collecting orbital, operational and organisational mission detail.", Segment: SegmentGround},
	{ID: "RD-0001", TacticID: TacticResourceDeployment, Name: "Acquire Infrastructure", Description: "Obtaining ground station or radio infrastructure to support later access.", Segment: SegmentGround},
	{ID: "IA-0002", TacticID: TacticInitialAccess, Name: "Compromise Software Defined Radio", Description: "Taking control of the SDR or its host in the ground segment.", Segment: SegmentGround},
	{ID: "IA-0007", TacticID: TacticInitialAccess, Name: "Compromise Ground System", Description: "Gaining access to mission ground systems, operator workstations or services.", Segment: SegmentGround},
	{ID: "IA-0008", TacticID: TacticInitialAccess, Name: "Rogue External Entity", Description: "An unauthorised ground station or spacecraft interacting with the mission.", Segment: SegmentLink},
	{ID: "IA-0009", TacticID: TacticInitialAccess, Name: "Trusted Relationship", Description: "Abusing a trusted partner, supplier or federated system to reach the mission.", Segment: SegmentGround},
	{ID: "EX-0001", TacticID: TacticExecution, Name: "Replay", Description: "Re-sending previously captured valid commands or link traffic.", Segment: SegmentLink},
	{ID: "EX-0009", TacticID: TacticExecution, Name: "Exploit Code Flaws", Description: "Exploiting software weaknesses in flight or ground software.", Segment: SegmentSpace},
	{ID: "EX-0012", TacticID: TacticExecution, Name: "Modify On-Board Values", Description: "Changing onboard parameters, tables or thresholds through commanding.", Segment: SegmentSpace},
	{ID: "EX-0013", TacticID: TacticExecution, Name: "Flooding", Description: "Saturating a mission interface with valid or malformed traffic.", Segment: SegmentLink},
	{ID: "EX-0014", TacticID: TacticExecution, Name: "Spoofing", Description: "Injecting traffic that appears to originate from a trusted party.", Segment: SegmentLink},
	{ID: "EX-0016", TacticID: TacticExecution, Name: "Jamming", Description: "Radiating energy that prevents the mission from closing its link.", Segment: SegmentLink},
	{ID: "PER-0002", TacticID: TacticPersistence, Name: "Backdoor", Description: "Leaving hardware or software access in place for reuse.", Segment: SegmentGround},
	{ID: "PER-0003", TacticID: TacticPersistence, Name: "Ground System Presence", Description: "Maintaining an unauthorised presence in the ground segment.", Segment: SegmentGround},
	{ID: "PER-0004", TacticID: TacticPersistence, Name: "Replace Cryptographic Keys", Description: "Substituting mission keys so adversary access survives rotation.", Segment: SegmentGround},
	{ID: "PER-0005", TacticID: TacticPersistence, Name: "Valid Credentials", Description: "Reusing legitimate credentials or certificates to retain access.", Segment: SegmentGround},
	{ID: "DE-0002", TacticID: TacticDefenseEvasion, Name: "Prevent Downlink", Description: "Stopping telemetry from reaching operators, including by jamming the link.", Segment: SegmentLink},
	{ID: "DE-0004", TacticID: TacticDefenseEvasion, Name: "Masquerading", Description: "Presenting adversary activity as legitimate mission activity.", Segment: SegmentGround},
	{ID: "DE-0010", TacticID: TacticDefenseEvasion, Name: "Overflow Audit Log", Description: "Flooding or truncating audit records so activity is not retained.", Segment: SegmentGround},
	{ID: "LM-0003", TacticID: TacticLateralMovement, Name: "Constellation Hopping via Crosslink", Description: "Reaching a second spacecraft through an inter-satellite link.", Segment: SegmentSpace},
	{ID: "LM-0007", TacticID: TacticLateralMovement, Name: "Valid Credentials", Description: "Using legitimate credentials to move between mission systems.", Segment: SegmentGround},
	{ID: "EXF-0003", TacticID: TacticExfiltration, Name: "Eavesdropping", Description: "Capturing mission data from the uplink or downlink.", Segment: SegmentLink},
	{ID: "EXF-0007", TacticID: TacticExfiltration, Name: "Compromised Ground System", Description: "Removing mission data through a compromised ground system.", Segment: SegmentGround},
	{ID: "IMP-0002", TacticID: TacticImpact, Name: "Disruption", Description: "Interrupting mission service for a period of time.", Segment: SegmentSpace},
	{ID: "IMP-0003", TacticID: TacticImpact, Name: "Denial", Description: "Preventing the mission from delivering its service.", Segment: SegmentSpace},
	{ID: "IMP-0004", TacticID: TacticImpact, Name: "Degradation", Description: "Reducing mission performance below its operational baseline.", Segment: SegmentSpace},
}

func Tactics() []Tactic {
	out := make([]Tactic, len(tactics))
	copy(out, tactics)
	sort.Slice(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}

func Techniques() []Technique {
	out := make([]Technique, len(techniques))
	copy(out, techniques)
	sort.Slice(out, func(i, j int) bool {
		if out[i].TacticID == out[j].TacticID {
			return out[i].ID < out[j].ID
		}
		return tacticOrder(out[i].TacticID) < tacticOrder(out[j].TacticID)
	})
	return out
}

func TechniqueByID(id string) (Technique, bool) {
	for _, technique := range techniques {
		if technique.ID == id {
			return technique, true
		}
	}
	return Technique{}, false
}

func TacticByID(id string) (Tactic, bool) {
	for _, tactic := range tactics {
		if tactic.ID == id {
			return tactic, true
		}
	}
	return Tactic{}, false
}

func tacticOrder(id string) int {
	for _, tactic := range tactics {
		if tactic.ID == id {
			return tactic.Order
		}
	}
	return len(tactics) + 1
}
