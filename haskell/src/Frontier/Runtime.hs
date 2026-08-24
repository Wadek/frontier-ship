-- | Runtime (R): post-ship probe and bounded chaos.
-- English: english/R_RUNTIME.md
module Frontier.Runtime
  ( RuntimeMode(..)
  , chaosAllowed
  , loopbackURL
  ) where

data RuntimeMode = Scan | ChaosDry | ChaosInject | Budget
  deriving (Eq, Show)

-- | Inject only when the allowlist says chaos is on, the operator
-- set the inject flag, and the target is allowlisted.
chaosAllowed :: Bool -> Bool -> Bool -> Bool
chaosAllowed allowlisted chaosEnabled injectFlag =
  allowlisted && chaosEnabled && injectFlag

-- | v1 HTTP targets must be loopback.
loopbackURL :: String -> Bool
loopbackURL u =
  take 16 u == "http://127.0.0.1"
    || take 17 u == "https://127.0.0.1"
    || take 16 u == "http://localhost"
    || take 17 u == "https://localhost"
