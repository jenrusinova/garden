
import './App.css';
import React from 'react';
import ReactDOM from 'react-dom';
import $ from 'jquery';
import logo from './images/gardenpic.jpg';
import ZoneList from './ZoneList.js';
import ZoneListEntry from './ZoneListEntry.js';

class App extends React.Component {
  constructor(props){
    super(props);
    this.state = {
      isZoneLoaded:false,
      status: true,
      zones:[]
    }
    this.handleZoneLoader = this.handleZoneLoader.bind(this);
    this.handleActiveZone = this.handleActiveZone.bind(this);
    this.handleTitleClick = this.handleTitleClick.bind(this);
    this.handleRuntimeButtonCLick = this.handleRuntimeButtonCLick.bind(this);
  }

  handleZoneLoader(){
  this.setState({
    isZoneLoaded:!this.state.isZoneLoaded
  })
  }
  handleRuntimeButtonCLick(buttonId, zone){
    let zones = this.state.zones;
    let objectToSend = {};
    console.log('button ' + buttonId + ' was clicked');
    let time = Number(buttonId);
    for (let i = 0; i < zones.length; i++ ){
      if(zone.name === zones[i].name){
        zones[i].runtime = time;
      }
    }
    console.log(this.state.zones);
    objectToSend['id'] = zone.id;
    objectToSend['runtime'] = zone.runtime;
    console.log(objectToSend);
    $.ajax({
      url: "http://localhost:8089/start/" + zone.id + "?time=" + time,
      type: 'GET',
      //contentType: 'application/json',
      //data:JSON.stringify(objectToSend),
      success: function(){
        console.log('sent POST successfully');
        this.setState({
          zones: this.state.zones
        })
        console.log(this.state);
      }.bind(this),
      error: function(err){
        console.log('error sending POST request', err);
        console.log("http://localhost:8089/start/" + zone.id + "?time=" + time,);
      }
  });



      //url:


  }

  handleTitleClick(zone){
    console.log(zone.name, ' was clicked');
    let zoneId = zone.id;
    let zones = this.state.zones;
    let objectToSend = {};
    const enteredZoneName = prompt('Please enter new zone name');
    for (let i = 0; i < zones.length; i++){
      console.log('inside for loop');
      if (zoneId === zones[i].id){
        zones[i].name = enteredZoneName;
      }
    }
    objectToSend['id'] = zone.id;
    objectToSend['name'] = zone.name;
    $.ajax({
      url: "/update/roses",
      type: 'POST',
      contentType: 'application/json',
      data:JSON.stringify(objectToSend),
      success: function(data){
        console.log('sent POST successfully');
        this.setState({
          zones: this.state.zones
        })
        console.log(this.state);
      }.bind(this),
      error: function(err){
        console.log('error sending POST request', err);
      }
  });



  }

  handleActiveZone(zone){
    let zones = this.state.zones;
    let zoneId = zone.id;
    var objectToSend = {};
    for (let i = 0; i < zones.length; i++){
      if(zones[i].id === zoneId){
        console.log('Clicked on', zones[i].id );
        zones[i].is_on = !zones[i].is_on;
        objectToSend['id'] = zones[i].id;
        objectToSend['is_on'] = zones[i].is_on;
      }
    }
  console.log(objectToSend);

    $.ajax({
      url: "/update/roses",
      type: 'POST',
      contentType: 'application/json',
      data:JSON.stringify(objectToSend),
      success: function(data){
        console.log('sent POST successfully');
        this.setState({
          zones: this.state.zones
        })
        console.log(this.state);
      }.bind(this),
      error: function(err){
        console.log('error sending POST request', err);
      }
  });
  }




  componentDidMount = () =>{
    $.ajax({
      url: "/zone",
      type: 'GET',
      dataType: 'json',
      success: function(data){
        console.log('sent GET successfully');
        this.setState({
          zones: data.zones
        })
        console.log(this.state);
      }.bind(this),
      error: function(err){
        console.log('error sending GET request', err);
      }
  });
  }



  renderView(){
    if(this.state.isZoneLoaded){
      return  <ZoneList zones = {this.state.zones}
               handleClick = {this.handleActiveZone}
               handleTitleClick = {this.handleTitleClick}
               handleRuntimeButtonCLick = {this.handleRuntimeButtonCLick}
      />
    }
  }

  render() {
    return (
      <div className='main'>
      <div className="logo">
      <img src={logo} width="25%" height="25%" alt="logo"/>
      </div>
         <div className = 'zone-button'>
         <button id="load-button" onClick = {this.handleZoneLoader}>
        Load zones
         </button>
         </div>

          <div className='zone-list'>
           {this.renderView()}
          </div>
          </div>

    );

  }
}

ReactDOM.render(
  <App />,
  document.getElementById('root')
);


export default App;
