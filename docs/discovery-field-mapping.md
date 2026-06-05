# Discovery → Mesh field mapping

This document maps every field `control` currently reads from the Hydden "discovery" API
(`control/backend/internal/features/integrations/hydden/mapping_*.go`) onto the mesh catalog
entities the `mesh-discovery` connector emits. It is the spec for the account / group / owner
collectors.

## Target structures (mesh-sdk v0.2.75)

| Concern | Definition entity | Edge entity | Edge carries |
|---|---|---|---|
| Free-form fields | `Attribute{AttributeRef,Name,Description}` | `AccountAttribute` / `GroupAttribute` / `PersonAttribute` `{…Ref, AttributeRef, Value, Type}` | value + type hint |
| Risk | `RiskFactor{RiskFactorRef,Name,Category,Weight,Description}` | `AccountRiskFactor{AccountRef,RiskFactorRef,Confidence}` | confidence |
| Classification | `Classification{ClassificationRef,Name,Description}` | `AccountClassification{AccountRef,ClassificationRef,Confidence}` | confidence |

**Hard SDK constraint:** `RiskFactor` and `Classification` only have **account** edges. There is no
`GroupRiskFactor`, `PersonClassification`, etc. Attribute edges exist for account / group / person
(but **not** role). Therefore:

- **Accounts** → typed columns + risk factors + classifications + attributes.
- **Groups** → typed columns + attributes only. Group "classification"/"critical"/"high-priv" and
  member-risk aggregates land as `GroupAttribute`.
- **Owners (Person)** → typed columns + attributes only.

**Confidence is always `1.0`** for both risk factors and classifications — discovery asserts these
directly, so there is no probabilistic component.

**Weight** on a `RiskFactor` definition = the numeric severity Hydden ships for that indicator
(`boolIndicator` reads a positive numeric weight; presence ⇒ true). Weight is a property of the
*definition*; per-account presence is the edge.

---

## ACCOUNT

### → typed `Account` columns

| Discovery field | Account column |
|---|---|
| `Id` (fallback `Email`) | `AccountRef` |
| `Account Type` | `AccountType` |
| `Account Name` | `Name` |
| `Display Name` | `DisplayName` |
| `UPN` | `UPN` |
| `Email` | `PrimaryEmail` |
| `Status` | `Status` (+ derive `Enabled`) |
| `Created` | `CreatedAt` |
| `Last Logon` | `LastLoginDate` |
| `Disabled Time` | `DisabledDate` |

> Lean on `connectorutil.NormalizeAttributes` — build one attribute list from the whole row, fold
> the keys that match typed columns onto the entity, emit the remainder as `AccountAttribute`.

### → `RiskFactor` + `AccountRiskFactor` (confidence 1.0)

Every `boolIndicator` field becomes one `RiskFactor` definition (emitted once) plus a per-account
edge when present. Category is taken from Hydden's own risk buckets (the `(Total)` groupings).

| Discovery field (`boolIndicator`) | RiskFactor `Category` |
|---|---|
| `Accounts with MFA Not Enabled` | Password & Security |
| `Accounts with Password 90+ Days` | Password & Security |
| `Accounts with Password 180+ Days` | Password & Security |
| `Accounts with Password 365+ Days` | Password & Security |
| `Accounts with Password Never Set` | Password & Security |
| `Account Password Not Changed Since Public Breach` | Password & Security |
| `Accounts not used in 90+ Days` | Account Activity |
| `Accounts not used in 180+ Days` | Account Activity |
| `Accounts not used in 365+ Days` | Account Activity |
| `Accounts with 10+ Failed Login Attempts in 1 Hour` | Account Activity |
| `Shared Accounts` | Account Statistics |
| `Account Group Deviation` | Group Membership |
| `Accounts with No Owner` | Owner Mapping |
| `Inactive Owners With Enabled Accounts` | Owner Mapping |
| `Privileged Accounts Not Vaulted` | Privilege |
| `Breached Account(s)` | Breach Data |

`RiskFactorRef` = a stable slug per indicator (e.g. `mfa-not-enabled`). `Weight` = Hydden's numeric
value for the indicator. `Description` = the human label.

### → `Classification` + `AccountClassification` (confidence 1.0)

| Source | Classification(s) |
|---|---|
| `Classifications` (split the list/CSV) | one `Classification` per tag (admin, sensitive, …) |
| `Is Privileged` == true | `Privileged` |
| `Is Global Admin` / `Global Admin` == true | `Global Admin` |
| `Is Main Account` == true | `Main Account` |

> **Decision point:** the privilege booleans (`Is Privileged`, `Is Global Admin`, `Is Main Account`)
> are modelled as classifications here because they classify the identity. The alternative is to
> leave them as plain attributes. The derived `PrivilegedAccount` (computed from `Classifications`
> containing "admin") is redundant with the tag list and is dropped.

### → `AccountAttribute` (everything remaining)

- **Profile/identity:** `Department`, `Title`, `Domain`, `Path`, `Computer Name`, `Home Dir`,
  `Login Shell`, `Employee ID`, `SAM Account Name`, `Manager`, `Manager ID`, `Manager EID`, `Glcode`.
- **Platform:** `Account Platform`, `Provider`.
- **Security state:** `MFA` (status), `Password Age`, `Password Changed` (datetime), `Login Age`.
- **Scores/aggregates:** `Total Threat`, `Risk Score` (int), `Account Activity (Total)`,
  `Account Statistics (Total)`, `Breach Data (Total)`, `Group Membership (Total)`,
  `Owner Mapping (Total)`, `Password & Security (Total)`, `Privilege (Total)`, `Actions`,
  `Expired Accounts (Aggregated)`.
- **Compromise detail:** `Compromise Age`, `Compromise Date`, `Compromise Name`.
- **PAM/vault:** `PAM Status` (text), `Managed by PAM`, `CyberArk Discovery`, `Safe Names`,
  `Safe Account Id`, `Safe Account Name`, `Safe Secret Types`, `Secret Type`, `Safes` (count),
  `Vault Id`.
- **Privilege detail (lists):** `Highly Privileged Group(s)`, `Highly Privileged Role(s)`,
  `All Privileged Groups`, `Elevations`. (Memberships themselves are captured as edges; these
  free-text lists are retained as attributes for fidelity.)
- **Ownership:** `Mapped To`, `Mapped Owners` (count).
- **Custom:** `Custom 1` … `Custom 10`.

> `Data Source Id` / `Data Source Name` / `Data Source Platform` are **not** attributes — they
> define the `Application` (datasource) the account is linked to and the connector scope.

---

## GROUP (typed columns + `GroupAttribute` only)

### → typed `Group` columns
| Discovery field | Group column |
|---|---|
| `Group Id` (fallback `Id`) | `GroupRef` |
| `Group Name` | `Name` |
| `Description` | `Description` |

(No group-created timestamp is collected today, so `CreatedAt` stays nil.)

### → `GroupAttribute`
`Group Display Name`, `Group Domain`, `Group Path`, `Group Computer Name`, `Owner`,
`Direct Member Count`, `Expanded Member Count`, `Classification`, `Is Critical`,
`Is High Privilege`, `Last Review Date`, `Next Review Date`, `Compliance Status`, `Total Threat`,
all `(Total)` aggregates, and the member-risk aggregates (`Account Group Deviation`,
`Account Password Not Changed Since Public Breach`, `Accounts not used in 90/180/365+ Days`,
`Accounts with 10+ Failed Login Attempts in 1 Hour`).

> **Decision point:** `Classification`, `Is Critical`, `Is High Privilege` are genuinely
> classification-like but have no group classification edge in the SDK, so they are attributes for
> now. If group-level classification/risk becomes a first-class need, the follow-up is to add
> `GroupClassification` / `GroupRiskFactor` to the SDK.

---

## OWNER → `Person` (typed columns + `PersonAttribute` only)

### → typed `Person` columns
| Discovery field | Person column |
|---|---|
| `Identity Id` (fallback `Id`) | `PersonRef` |
| `Identity Name` | `Name` |
| `Identity Email` (fallback `Email`) | `PrimaryEmail` |

### → `PersonAttribute`
`Alt Email`, `Phone`, `Mobile Phone`, `Department`, `Title`, `Manager`, `Location`, `Status`,
`Owner Type`, `Start Date`, `End Date`, `Roles` (list), `Total Threat`, all `(Total)` aggregates,
and the owned-account risk aggregates (`Account Group Deviation`, `Account Breached`,
`Accounts not used in 90/180/365+ Days`, `Accounts with 10+ Failed Login Attempts in 1 Hour`,
`Accounts with MFA Not Enabled`, `Accounts with No Owner`).

> Owner↔account linkage is an edge (owner mapping), not an attribute. `Owner Type` has no person
> classification edge → attribute.

---

## ROLE (out of scope for this attribute pass)

The application-role collector emits `Role` + `AccountRole`. Extra role fields (`Path`,
`Role Platform`, `Direct/Expanded Role Count`) currently have **no attribute home** — the SDK has
no `RoleAttribute`. Left as a follow-up if role enrichment is required.

---

## Collector ownership

The **account collector** emits the account-scoped graph: `Account`, `AccountAttribute`, plus the
`Attribute` / `RiskFactor` / `Classification` definitions and the `AccountRiskFactor` /
`AccountClassification` edges. The **group** and **owner** collectors emit their entity + their
attribute value edges (`GroupAttribute` / `PersonAttribute`).

**The `Attribute` definition dictionary (`attributes` space) is owned solely by the account
collector.** mesh-sdk allows one collector per space, and — more importantly — two collectors
writing the same space would merkle-prune each other's entries on every sync. So group and owner
attributes emit value edges only; their `attribute_ref`s are self-describing keys (a definition
entry is created when the same key also appears on an account). See `emitNamedAttributes`.

**Attribute sources.** Account and group attributes come from two places: the discovery grid (the
computed/enriched columns) **and** the per-datasource datastore fetch
(`/internal/v1/datastore/fetch`, the native source-system object). Owners/identities are **not**
per-datasource datastore entities (the datastore only holds `principal.account.*` and `edge.*`
types), so person attributes are sourced from the owner feed — the full owner object — using the
same flatten/emit pattern. If discovery later exposes identities through the datastore fetch, the
owner collector can add that pass like the account/group collectors.

## Typed Account columns now mapped

`Status` (→ `AccountStatus` enum, plus the existing `Enabled`), `UPN`, `CreatedAt` (Created),
`LastLoginDate` (Last Logon), and `DisabledDate` (Disabled Time) are now folded onto the typed
`Account` entity rather than left as attributes.
