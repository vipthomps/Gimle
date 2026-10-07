import About from "../components/Config/About"
import Backups from "../components/Config/Backups"
import Basic from "../components/Config/Basic"
import Connectors from "../components/Config/Connectors"
import Docs from "../components/Config/Docs"
import Influx from "../components/Config/Influx"
import Names from "../components/Config/Names"
import PortScan from "../components/Config/PortScan"
import Prometheus from "../components/Config/Prometheus"
import Scan from "../components/Config/Scan"

function Config() {

  return (
    <div class="row">
      <div class="col-md">
        
        <Basic></Basic>
        
        <div class="mt-4">
          <Scan></Scan>
        </div>
        <div class="mt-4">
          <Names></Names>
        </div>
        <div class="mt-4 mb-4">
          <Backups></Backups>
        </div>
      </div>
      <div class="col-md">
        
        <Connectors></Connectors>

        <div class="mt-4">
          <Docs></Docs>
        </div>

        <div class="mt-4">
          <PortScan></PortScan>
        </div>

        <div class="mt-4">
          <Influx></Influx>
        </div>
        
        <div class="mt-4">
          <Prometheus></Prometheus>
        </div>
        <div class="mt-4">
          <About></About>
        </div>
      </div>
    </div>
  )
}

export default Config