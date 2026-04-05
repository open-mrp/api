#!/usr/bin/env python3
"""
Domain name brainstorming and availability checker.

Generates dictionary words relevant to an ERP automation / autonomous agent platform,
then checks .com and .ai availability via DNS + WHOIS.
"""

import subprocess
import socket
import time
import sys

# ──────────────────────────────────────────────────────────────────────
# Word list: curated dictionary words that evoke automation, agents,
# orchestration, operations, manufacturing, logistics, autonomy, etc.
# ──────────────────────────────────────────────────────────────────────

CANDIDATES = [
    # Autonomy / agency
    "autonoma",
    "automata",
    "automate",
    "autonomy",
    "sentient",
    "agentic",
    "agentive",
    "delegate",
    "emissary",
    "envoy",
    "proxy",
    "steward",
    "warden",
    "custodian",
    "prefect",
    "regent",
    "viceroy",
    "deputy",
    "adjutant",
    "marshal",
    "sentinel",
    "sentry",

    # Orchestration / flow / process
    "orchestrate",
    "cadence",
    "tempo",
    "rhythm",
    "sequence",
    "cascade",
    "conduit",
    "nexus",
    "relay",
    "dispatch",
    "convoy",
    "manifold",
    "pipeline",
    "circuit",
    "loom",
    "weave",
    "lattice",
    "matrix",
    "scaffold",
    "trellis",
    "arbor",

    # Operations / manufacturing / ERP
    "foundry",
    "forge",
    "anvil",
    "crucible",
    "kiln",
    "lathe",
    "spindle",
    "shuttle",
    "fulcrum",
    "lever",
    "piston",
    "crank",
    "pulley",
    "gantry",
    "derrick",
    "hoist",
    "winch",
    "capstan",
    "turbine",
    "dynamo",
    "rotor",
    "armature",
    "camshaft",

    # Intelligence / cognition
    "cognition",
    "acumen",
    "sapient",
    "lucid",
    "cortex",
    "synapse",
    "neuron",
    "axiom",
    "theorem",
    "calculus",
    "quorum",

    # Movement / logistics
    "vector",
    "vertex",
    "traverse",
    "meridian",
    "azimuth",
    "bearing",
    "heading",
    "waypoint",
    "beacon",
    "compass",
    "rudder",
    "helm",
    "tiller",
    "keel",
    "ballast",

    # Structure / foundation
    "bastion",
    "citadel",
    "rampart",
    "bulwark",
    "keystone",
    "capstone",
    "lintel",
    "plinth",
    "pediment",
    "cornice",
    "buttress",
    "pillar",
    "monolith",
    "obelisk",
    "spire",
    "pinnacle",
    "apex",
    "zenith",
    "summit",
    "crest",

    # Power / energy / force
    "catalyst",
    "ignite",
    "kinetic",
    "voltage",
    "ampere",
    "tesla",
    "watt",
    "joule",
    "flux",
    "surge",
    "pulse",
    "torque",
    "thrust",
    "impulse",
    "momentum",
    "inertia",

    # Command / control
    "mandate",
    "edict",
    "decree",
    "charter",
    "protocol",
    "doctrine",
    "canon",
    "codex",
    "ledger",
    "register",
    "manifest",
    "dossier",
    "brief",

    # Precision / craft
    "caliber",
    "gauge",
    "mitre",
    "chisel",
    "scribe",
    "stencil",
    "template",
    "jig",
    "clamp",
    "vice",

    # Nature-inspired (strength, reliability)
    "granite",
    "obsidian",
    "flint",
    "cobalt",
    "iron",
    "steel",
    "bronze",
    "alloy",
    "carbon",
    "silicon",
    "titanium",
    "tungsten",
    "osmium",
    "rhodium",
    "iridium",
    "platinum",
    "adamant",
    "onyx",

    # Abstract / evocative
    "praxis",
    "ethos",
    "logos",
    "telos",
    "kairos",
    "pragma",
    "ergon",
    "opus",
    "verve",
    "vigor",
    "mettle",
    "grit",
    "nerve",
    "resolve",
    "prowess",

    # Short punchy names
    "aeon",
    "arc",
    "axis",
    "bolt",
    "core",
    "crux",
    "dash",
    "edge",
    "flux",
    "glyph",
    "grid",
    "hive",
    "hub",
    "ion",
    "jolt",
    "knot",
    "link",
    "mesh",
    "node",
    "orb",
    "pact",
    "rift",
    "rune",
    "sync",
    "tier",
    "unit",
    "vane",
    "volt",
    "weld",
    "zeal",
    "zinc",

    # Two-syllable strong brand words
    "anvil",
    "cipher",
    "daemon",
    "falcon",
    "galvan",
    "harbor",
    "jackal",
    "kernel",
    "mantle",
    "optic",
    "parcel",
    "quartz",
    "ratchet",
    "signet",
    "talon",
    "valiant",
    "warrant",

    # Compound-ish / invented but dictionary-adjacent
    "ironclad",
    "clockwork",
    "flywheel",
    "mainspring",
    "bedrock",
    "lodestone",
    "loadstar",
    "lodestar",
    "workhorse",
    "powerhouse",
    "stronghold",
    "watchtower",
    "wheelhouse",
    "cornerstone",
    "underpinning",
    "juggernaut",
    "vanguard",
    "trailblaze",
    "pathfinder",
    "wayfinder",
    "taskmaster",
    "gatekeeper",
    "spearhead",
    "benchmark",
    "watershed",
    "breakwater",
    "headway",
    "foothold",
]

# Deduplicate (case-insensitive) while preserving order
seen = set()
UNIQUE_CANDIDATES = []
for w in CANDIDATES:
    key = w.lower()
    if key not in seen:
        seen.add(key)
        UNIQUE_CANDIDATES.append(key)


def check_dns(domain: str) -> bool:
    """Return True if the domain resolves (likely registered)."""
    try:
        socket.setdefaulttimeout(3)
        socket.getaddrinfo(domain, None)
        return True
    except (socket.gaierror, socket.timeout, OSError):
        return False


def check_whois(domain: str) -> bool:
    """
    Return True if whois suggests the domain is registered.
    Falls back to DNS if whois is unavailable.
    """
    try:
        result = subprocess.run(
            ["whois", domain],
            capture_output=True,
            text=True,
            timeout=10,
        )
        output = result.stdout.lower()

        # Common "not found" indicators
        not_found_signals = [
            "no match for",
            "not found",
            "no entries found",
            "no data found",
            "domain not found",
            "status: free",
            "status: available",
            "no object found",
            "nothing found",
            "is available for registration",
        ]
        for signal in not_found_signals:
            if signal in output:
                return False

        # If whois returned substantial output with registrar info, it's taken
        registered_signals = [
            "registrar:",
            "creation date:",
            "created:",
            "registry domain id:",
            "domain name:",
            "name server:",
            "nserver:",
        ]
        for signal in registered_signals:
            if signal in output:
                return True

        # Ambiguous - fall back to DNS
        return check_dns(domain)

    except (subprocess.TimeoutExpired, FileNotFoundError):
        return check_dns(domain)


def main():
    tlds = [".com", ".ai"]
    total_checks = len(UNIQUE_CANDIDATES) * len(tlds)

    print(f"Checking {len(UNIQUE_CANDIDATES)} words across {len(tlds)} TLDs ({total_checks} lookups)...")
    print("This will take a few minutes to be respectful of rate limits.\n")

    results = {}  # domain -> bool (True = taken)

    for i, word in enumerate(UNIQUE_CANDIDATES):
        for tld in tlds:
            domain = f"{word}{tld}"
            sys.stdout.write(f"\r[{i * len(tlds) + tlds.index(tld) + 1}/{total_checks}] Checking {domain}...          ")
            sys.stdout.flush()

            taken = check_whois(domain)
            results[domain] = taken

            # Small delay to avoid hammering whois servers
            time.sleep(0.8)

    # Clear the progress line
    sys.stdout.write("\r" + " " * 80 + "\r")

    # ── Sort and display results ──────────────────────────────────
    available = {d: t for d, t in results.items() if not t}
    taken = {d: t for d, t in results.items() if t}

    print("=" * 64)
    print(f"  AVAILABLE DOMAINS ({len(available)})")
    print("=" * 64)

    # Group by word
    available_by_word = {}
    for domain in sorted(available.keys()):
        word = domain.rsplit(".", 1)[0]
        available_by_word.setdefault(word, []).append(domain)

    for word in sorted(available_by_word.keys()):
        domains = available_by_word[word]
        print(f"  {word:20s}  ->  {', '.join(domains)}")

    print()
    print("=" * 64)
    print(f"  TAKEN DOMAINS ({len(taken)}) — potential acquisition targets")
    print("=" * 64)

    taken_by_word = {}
    for domain in sorted(taken.keys()):
        word = domain.rsplit(".", 1)[0]
        taken_by_word.setdefault(word, []).append(domain)

    for word in sorted(taken_by_word.keys()):
        domains = taken_by_word[word]
        print(f"  {word:20s}  ->  {', '.join(domains)}")

    # ── Summary ───────────────────────────────────────────────────
    print()
    print("-" * 64)

    # Words where BOTH .com and .ai are available
    both_available = []
    for word in sorted(set(d.rsplit(".", 1)[0] for d in results)):
        com = f"{word}.com"
        ai = f"{word}.ai"
        if com in results and ai in results:
            if not results[com] and not results[ai]:
                both_available.append(word)

    if both_available:
        print(f"\n  Words with BOTH .com AND .ai available ({len(both_available)}):")
        for w in both_available:
            print(f"    {w}")

    print()


if __name__ == "__main__":
    main()
