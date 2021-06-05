
import './App.css';
import React from 'react';
import ReactDOM from 'react-dom';
import $ from 'jquery';
import logo from './images/grow_dammit.jpg'

class App extends React.Component {


  componentDidMount(){
    $.ajax({
      url: "/zone",
      type: 'GET',
      dataType: 'json', // added data type
      success: function(res) {
          console.log(res);

      }
  });
  }

  render() {
    return (
      <div className="logo">
      <img src={logo} width="25%" height="25%" alt="logo"/>
         </div>
    );


  }
}

ReactDOM.render(
  <App />,
  document.getElementById('root')
);


export default App;
