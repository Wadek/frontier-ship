-- Frontier.Monitor — pure witness of the monitor replay rules.
--
-- English: english/MONITOR.md
-- Go:      internal/monitor/monitor.go (audits real ledger files)
--
-- This module states, in pure form, the directive checks D1-D5 over the
-- ledger: push.authorized needs a gate.passed that follows a plan.passed
-- for the same branch+HEAD; no soft bypass; no ship from main/master; no
-- gate while a blocking exam stands; denied attempts are incidents (watch).
--
-- Two things stay in the Go runtime on purpose, mirroring Gate.hs:
--   D0 cryptographic chain integrity (base has no SHA-256), and
--   time-based freshness (policy.GateTTL — the pure layer models order and
--   state; the runtime measures the 15-minute window from row timestamps).

module Frontier.Monitor
  ( MSeal (..),
    MVerdict (..),
    monitorAudit,
  )
where

import Data.List (lookup)

-- | One sealed ledger row, reduced to the fields the replay needs.
data MSeal = MSeal
  { mAction :: String, -- plan.passed | gate.passed | push.authorized | ...
    mBranch :: String,
    mHead :: String,
    mBlocked :: Bool -- exam.owasp blocks_gate flag
  }
  deriving (Eq, Show)

-- | Verdict, ordered by severity (higher = worse).
data MVerdict = MClean | MWatch | MViolation
  deriving (Eq, Ord, Show)

-- | One replay state slot per branch+HEAD pair.
data Slot = Slot
  { planAt :: Maybe Bool, -- fresh plan.passed present (Just True)
    gateAt :: Maybe Bool, -- fresh gate.passed present (Just True)
    examBlocked :: Bool
  }
  deriving (Eq, Show)

emptySlot :: Slot
emptySlot = Slot Nothing Nothing False

-- | Replay one row over the state; returns (findings, updated state).
step :: MSeal -> Slot -> ([(String, String)], Slot)
step s st =
  case mAction s of
    "plan.passed" ->
      ( mainFinding,
        st {planAt = Just True}
      )
    "gate.passed" ->
      ( mainFinding
          ++ [ ("D1", "gate.passed without a fresh plan.passed for this branch+HEAD")
               | planAt st /= Just True
             ]
          ++ [ ("D4", "gate.passed while a blocking exam.owasp stands for this branch+HEAD")
               | examBlocked st
             ],
        st {gateAt = Just True}
      )
    "push.authorized" ->
      ( mainFinding
          ++ [ ("D1", "push.authorized without a fresh gate.passed for this branch+HEAD")
               | gateAt st /= Just True
             ],
        st
      )
    "push.soft_allow" ->
      ( [("D3", "FRONTIER_SOFT=1 soft-allow of a would-be deny")],
        st
      )
    "exam.owasp" ->
      ( [], st {examBlocked = mBlocked s} )
    a
      | a `elem` ["plan.failed", "gate.failed", "push.deny", "apply.deny", "commit.deny_main"] ->
          ( [("D5", "denied attempt recorded: " ++ a)],
            st
          )
      | otherwise -> ([], st)
  where
    mainFinding =
      [ ("D2", "ship sealed from " ++ mBranch s)
        | isMainBranch (mBranch s)
      ]

isMainBranch :: String -> Bool
isMainBranch b = lower b `elem` ["main", "master"]
  where
    lower = map toLowerChar
    toLowerChar c
      | c >= 'A' && c <= 'Z' = toEnum (fromEnum c + 32)
      | otherwise = c

-- | Full replay of one ledger: (verdict, findings).
--   State is keyed by branch+HEAD, mirroring the Go runtime.
monitorAudit :: [MSeal] -> (MVerdict, [(String, String)])
monitorAudit rows = (verdictOf checks, checks)
  where
    (checks, _) = foldl stepAll ([], []) rows
    stepAll (acc, slots) s =
      let k = slotKey s
          st = maybe emptySlot id (lookup k slots)
          (findings, st') = step s st
          slots' = (k, st') : filter ((/= k) . fst) slots
       in (acc ++ findings, slots')
    slotKey s = (mBranch s, mHead s)
    verdictOf fs
      | any ((`elem` ["D1", "D2", "D3", "D4"]) . fst) fs = MViolation
      | any ((== "D5") . fst) fs = MWatch
      | otherwise = MClean
