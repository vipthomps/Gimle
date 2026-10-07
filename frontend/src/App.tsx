import { lazy, onMount } from 'solid-js';
import { Router, Route } from "@solidjs/router";
import './theme.css';
import './App.css';
import { runAtStart } from './functions/atstart';

import Body from './pages/Body';
import Header from './components/Header';

function App() {

  onMount(() => {
    runAtStart();
  });

  const Config = lazy(() => import("./pages/Config"));
  const History = lazy(() => import("./pages/History"));
  const HostPage = lazy(() => import("./pages/HostPage"));
  const Subnets = lazy(() => import("./pages/Subnets"));
  const SubnetPage = lazy(() => import("./pages/SubnetPage"));
  const MapPage = lazy(() => import("./pages/MapPage"));
  const ViewPage = lazy(() => import("./pages/ViewPage"));
  const StatsPage = lazy(() => import("./pages/StatsPage"));
  const StartPage = lazy(() => import("./pages/StartPage"));
  const BookmarksPage = lazy(() => import("./pages/BookmarksPage"));
  const AboutPage = lazy(() => import("./pages/AboutPage"));
  const DocsPage = lazy(() => import("./pages/DocsPage"));

  return (
    <div class="gm-shell">
    <Header></Header>
    <main class="gm-main">
          <Router>
            <Route path="/" component={StartPage}/>
            <Route path="/map" component={MapPage}/>
            <Route path="/bookmarks" component={BookmarksPage}/>
            <Route path="/view/:id" component={ViewPage}/>
            <Route path="/hosts" component={Body}/>
            <Route path="/config" component={Config}/>
            <Route path="/history" component={History}/>
            <Route path="/host/:id" component={HostPage}/>
            <Route path="/subnets" component={Subnets}/>
            <Route path="/stats" component={StatsPage}/>
            <Route path="/subnet/:id" component={SubnetPage}/>
            <Route path="/about" component={AboutPage}/>
            <Route path="/docs/*path" component={DocsPage}/>
          </Router>
    </main>
    </div>
  )
}

export default App
