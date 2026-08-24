-- | Hygiene (H): AI-provenance inspect/clean.
-- English: english/H_HYGIENE.md
-- Runtime talks to a local watermarks-remover HTTP service (Go).
module Frontier.Hygiene
  ( HygieneAction(..)
  , HygieneDisposition(..)
  , hygieneDisposition
  , blocksShip
  ) where

data HygieneAction = Inspect | Clean | Status
  deriving (Eq, Show)

data HygieneDisposition = HRecord | HAdvise | HBlock
  deriving (Eq, Show)

-- | Service-down or clean tree => record.
-- Marks present => advise, unless the operator asked to block.
hygieneDisposition :: Bool -> Int -> Bool -> HygieneDisposition
hygieneDisposition healthy suspicious blockFlag
  | blockFlag && healthy && suspicious > 0 = HBlock
  | healthy && suspicious > 0              = HAdvise
  | otherwise                              = HRecord

blocksShip :: HygieneDisposition -> Bool
blocksShip HBlock = True
blocksShip _      = False
