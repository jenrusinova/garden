
import './App.css';
import React from 'react';
import ReactDOM from 'react-dom';
import $ from 'jquery';
import logo from './images/grow_dammit.jpg'

class App extends React.Component {
  constructor(props){
    super(props);
    this.state = {
      isZoneLoaded:false,
      status: true,
      zones:[]
    }
    this.handleZoneLoader = this.handleZoneLoader.bind(this);
  }

  handleZoneLoader(){
  alert('Zone loaded');
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

  render() {
    return (
      <div className="logo">
      <img src={logo} width="25%" height="25%" alt="logo"/>
         <div className = 'zone-button'>
         <button onClick = {this.handleZoneLoader}>
        Load zones
         </button>
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
