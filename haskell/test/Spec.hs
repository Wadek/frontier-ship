module Main where

import Frontier.Gate
import Frontier.Hygiene
import Frontier.Laws
import Frontier.Monitor
import Frontier.Role
import Frontier.Runtime
import System.Exit (exitFailure, exitSuccess)

-- Monitor witnesses: rows reduced to the replay fields.
seal :: String -> String -> String -> Bool -> MSeal
seal a b h blocked = MSeal {mAction = a, mBranch = b, mHead = h, mBlocked = blocked}

-- A compliant plan -> gate -> push on a feature branch.
compliantRows :: [MSeal]
compliantRows =
  [ seal "plan.passed" "frontier/x" "h1" False,
    seal "gate.passed" "frontier/x" "h1" False,
    seal "push.authorized" "frontier/x" "h1" False
  ]

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
      -- Monitor (Frontier.Monitor)
      ok13 = monitorAudit compliantRows == (MClean, [])
      ok14 = fst (monitorAudit [seal "push.authorized" "frontier/x" "h1" False]) == MViolation
      ok15 = fst (monitorAudit [seal "push.soft_allow" "frontier/x" "h1" False]) == MViolation
      ok16 = fst (monitorAudit [seal "push.deny" "frontier/x" "h1" False]) == MWatch
      ok17 = fst (monitorAudit [seal "plan.passed" "main" "h1" False]) == MViolation
      ok18 = fst (monitorAudit
              [ seal "plan.passed" "frontier/x" "h1" False,
                seal "exam.owasp" "frontier/x" "h1" True,
                seal "gate.passed" "frontier/x" "h1" False
              ]) == MViolation
      ok19 = fst (monitorAudit [seal "gate.passed" "frontier/x" "h1" False]) == MViolation
      -- D4 clears after a clean re-exam of the same branch+HEAD.
      ok20 = let (v, _) = monitorAudit
               [ seal "plan.passed" "frontier/x" "h1" False,
                 seal "exam.owasp" "frontier/x" "h1" True,
                 seal "exam.owasp" "frontier/x" "h1" False,
                 seal "gate.passed" "frontier/x" "h1" False
               ]
             in v == MClean
  if and [ok1, ok2, ok3, ok4, ok5, ok6, ok7, ok8, ok9, ok10, ok11, ok12,
          ok13, ok14, ok15, ok16, ok17, ok18, ok19, ok20]
    then putStrLn "frontier-laws: ok" >> exitSuccess
    else putStrLn "frontier-laws: FAIL" >> exitFailure
