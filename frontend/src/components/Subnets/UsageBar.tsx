import { SubnetStat } from "../../functions/exports";

function UsageBar(props: { stat: SubnetStat }) {

  const pct = (n: number) => props.stat.Total ? (n * 100 / props.stat.Total) + "%" : "0%";

  return (
    <>
    <div class="progress-stacked" style="height: 0.9em;">
      <div class="progress" style={{width: pct(props.stat.Online)}} title={props.stat.Online+" online"}>
        <div class="progress-bar bg-success"></div>
      </div>
      <div class="progress" style={{width: pct(props.stat.Offline)}} title={props.stat.Offline+" seen, offline"}>
        <div class="progress-bar bg-secondary"></div>
      </div>
      <div class="progress" style={{width: pct(props.stat.Reserved)}} title={props.stat.Reserved+" reserved"}>
        <div class="progress-bar bg-warning"></div>
      </div>
    </div>
    <small class="opacity-75">{props.stat.Utilization}% used · {props.stat.Free} of {props.stat.Total} free</small>
    </>
  )
}

export default UsageBar
