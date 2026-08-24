module Main where

import Frontier.Gate
import Frontier.Hygiene
import Frontier.Laws
import Frontier.Role
import Frontier.Runtime
import System.Exit (exitFailure, exitSuccess)

main :: IO ()
main = do
  let ok1 = not (Observer `can` Operator)
      ok2 = Operator `can` Analyst
      ok3 = elevate Observer == Right Analyst
      ok4 = not $ ok $ evaluatePushGate GateInput
              { branch = "main", headSha = "abc", dirty = False, allowDirty = False }
      ok5 = ok $ evaluatePushGate GateInput
              { branch = "frontier/x", headSha = "abc", dirty = False, allowDirty = False }
      ok6 = F0 `higherThan` F4
      ok7 = hygieneDisposition True 1 False == HAdvise
      ok8 = not $ blocksShip $ hygieneDisposition False 3 True
      ok9 = loopbackURL "http://127.0.0.1:8765/health"
      ok10 = not $ loopbackURL "https://example.com/x"
      ok11 = not $ chaosAllowed True True False
      ok12 = chaosAllowed True True True
  if and [ok1, ok2, ok3, ok4, ok5, ok6, ok7, ok8, ok9, ok10, ok11, ok12]
    then putStrLn "frontier-laws: ok" >> exitSuccess
    else putStrLn "frontier-laws: FAIL" >> exitFailure
