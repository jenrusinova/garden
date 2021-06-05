
import './App.css';
import React from 'react';
import ReactDOM from 'react-dom';
import $ from 'jquery';

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
    return <h1>Loading...</h1>;
  }
}

ReactDOM.render(
  <App />,
  document.getElementById('root')
);


export default App;
